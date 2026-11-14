package runner

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/fblazt/boomerang/internal/models"
)

// EvaluateAssertions executes JavaScript-compatible test assertions (pm.test / pm.expect / pm.response)
// against req and resp and returns structured pass/fail results.
func EvaluateAssertions(script string, req models.RequestModel, resp *models.ResponseModel) []models.TestAssertionResult {
	script = stripJSComments(strings.TrimSpace(script))
	if script == "" {
		return nil
	}

	env := newScriptEnv(req, resp)
	var results []models.TestAssertionResult

	stmts := splitTopLevelStatements(script)
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if title, body, ok := parsePMTestCall(stmt); ok {
			res := models.TestAssertionResult{Name: title, Passed: true}
			if resp == nil {
				res.Passed = false
				res.Message = "no response available"
				results = append(results, res)
				continue
			}
			blockStmts := splitTopLevelStatements(body)
			for _, bs := range blockStmts {
				bs = strings.TrimSpace(bs)
				if bs == "" {
					continue
				}
				if handled := env.tryBindVar(bs); handled {
					continue
				}
				if err := env.evalAssertion(bs); err != nil {
					res.Passed = false
					res.Message = err.Error()
					break
				}
			}
			results = append(results, res)
			continue
		}

		if env.tryBindVar(stmt) {
			continue
		}

		res := models.TestAssertionResult{Name: summarizeStmt(stmt), Passed: true}
		if err := env.evalAssertion(stmt); err != nil {
			res.Passed = false
			res.Message = err.Error()
		}
		results = append(results, res)
	}

	return results
}

type scriptEnv struct {
	req      models.RequestModel
	resp     *models.ResponseModel
	jsonBody any
	jsonErr  error
	jsonOnce bool
	vars     map[string]any
}

func newScriptEnv(req models.RequestModel, resp *models.ResponseModel) *scriptEnv {
	return &scriptEnv{
		req:  req,
		resp: resp,
		vars: make(map[string]any),
	}
}

func (e *scriptEnv) parsedJSON() (any, error) {
	if !e.jsonOnce {
		e.jsonOnce = true
		if e.resp == nil || strings.TrimSpace(e.resp.Body) == "" {
			e.jsonErr = fmt.Errorf("response body is empty")
		} else {
			e.jsonErr = json.Unmarshal([]byte(e.resp.Body), &e.jsonBody)
		}
	}
	return e.jsonBody, e.jsonErr
}

func (e *scriptEnv) tryBindVar(stmt string) bool {
	trimmed := strings.TrimSpace(strings.TrimSuffix(stmt, ";"))
	for _, kw := range []string{"const ", "let ", "var "} {
		if strings.HasPrefix(trimmed, kw) {
			rest := strings.TrimSpace(trimmed[len(kw):])
			eq := strings.Index(rest, "=")
			if eq <= 0 {
				return false
			}
			name := strings.TrimSpace(rest[:eq])
			rhs := strings.TrimSpace(rest[eq+1:])
			val, err := e.resolveExpr(rhs)
			if err == nil {
				e.vars[name] = val
			} else {
				e.vars[name] = err
			}
			return true
		}
	}
	return false
}

func (e *scriptEnv) evalAssertion(stmt string) error {
	s := strings.TrimSpace(strings.TrimSuffix(stmt, ";"))
	if e.resp == nil {
		return fmt.Errorf("response is nil")
	}

	if strings.HasPrefix(s, "pm.response.to.") {
		return e.evalResponseTo(strings.TrimPrefix(s, "pm.response.to."))
	}
	if strings.HasPrefix(s, "pm.expect(") {
		return e.evalExpect(s)
	}
	if strings.HasPrefix(s, "assert(") && strings.HasSuffix(s, ")") {
		inner := s[len("assert(") : len(s)-1]
		args := splitCallArgs(inner)
		if len(args) == 0 {
			return fmt.Errorf("empty assert()")
		}
		ok, err := e.evalBooleanExpr(args[0])
		if err != nil {
			return err
		}
		if !ok {
			msg := "assertion failed"
			if len(args) > 1 {
				if m, merr := e.resolveExpr(args[1]); merr == nil {
					msg = fmt.Sprint(m)
				}
			}
			return fmt.Errorf("%s", msg)
		}
		return nil
	}

	ok, err := e.evalBooleanExpr(s)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("expression evaluated to false: %s", s)
	}
	return nil
}

func (e *scriptEnv) evalResponseTo(chain string) error {
	negated := false
	if strings.HasPrefix(chain, "not.") {
		negated = true
		chain = strings.TrimPrefix(chain, "not.")
	}
	var passed bool
	var desc string

	switch {
	case strings.HasPrefix(chain, "have.status(") && strings.HasSuffix(chain, ")"):
		argStr := chain[len("have.status(") : len(chain)-1]
		argVal, err := e.resolveExpr(argStr)
		if err != nil {
			return err
		}
		want, ok := toNumber(argVal)
		if !ok {
			return fmt.Errorf("status code argument must be numeric: %v", argVal)
		}
		got := float64(e.resp.StatusCode)
		passed = got == want
		desc = fmt.Sprintf("expected status %.0f, got %d", want, e.resp.StatusCode)

	case strings.HasPrefix(chain, "have.header(") && strings.HasSuffix(chain, ")"):
		args := splitCallArgs(chain[len("have.header(") : len(chain)-1])
		if len(args) == 0 {
			return fmt.Errorf("have.header requires header name")
		}
		nameVal, err := e.resolveExpr(args[0])
		if err != nil {
			return err
		}
		hName := fmt.Sprint(nameVal)
		hVal, exists := lookupHeaderCaseInsensitive(e.resp.Headers, hName)
		passed = exists
		desc = fmt.Sprintf("header %q missing", hName)
		if exists && len(args) > 1 {
			wantVal, err := e.resolveExpr(args[1])
			if err != nil {
				return err
			}
			wantStr := fmt.Sprint(wantVal)
			passed = strings.Contains(strings.ToLower(hVal), strings.ToLower(wantStr))
			desc = fmt.Sprintf("header %q value %q does not match %q", hName, hVal, wantStr)
		}

	case chain == "be.ok":
		passed = e.resp.StatusCode >= 200 && e.resp.StatusCode < 300
		desc = fmt.Sprintf("expected 2xx response, got %d", e.resp.StatusCode)

	default:
		return fmt.Errorf("unsupported pm.response.to matcher: %s", chain)
	}

	if negated {
		if passed {
			return fmt.Errorf("negated assertion failed (%s)", desc)
		}
		return nil
	}
	if !passed {
		return fmt.Errorf("%s", desc)
	}
	return nil
}

func (e *scriptEnv) evalExpect(expr string) error {
	openIdx := strings.Index(expr, "(")
	closeIdx := matchingParen(expr, openIdx)
	if closeIdx == -1 {
		return fmt.Errorf("unterminated pm.expect call")
	}
	targetExpr := strings.TrimSpace(expr[openIdx+1 : closeIdx])
	rest := strings.TrimSpace(expr[closeIdx+1:])
	rest = strings.TrimPrefix(rest, ".")

	actual, err := e.resolveExpr(targetExpr)

	// Strip optional ".to" / ".not" / ".be" / ".have" / ".at" / ".deep" prefixes carefully.
	negated := false
	tokens, callName, callArgs, parseErr := parseExpectChain(rest)
	if parseErr != nil {
		return parseErr
	}
	for _, tok := range tokens {
		if tok == "not" {
			negated = !negated
		}
	}

	// Existence checks can inspect missing JSON path cleanly.
	if callName == "exist" && len(callArgs) == 0 {
		exists := err == nil && actual != nil
		if negated {
			if exists {
				return fmt.Errorf("expected %s not to exist, got %v", targetExpr, actual)
			}
			return nil
		}
		if !exists {
			return fmt.Errorf("expected %s to exist", targetExpr)
		}
		return nil
	}

	if err != nil {
		return err
	}

	var passed bool
	var failMsg string

	switch callName {
	case "equal", "eql", "equals":
		if len(callArgs) < 1 {
			return fmt.Errorf("%s requires an expected value", callName)
		}
		want, werr := e.resolveExpr(callArgs[0])
		if werr != nil {
			return werr
		}
		passed = valuesEqual(actual, want)
		failMsg = fmt.Sprintf("expected %v to equal %v", actual, want)

	case "below", "lessThan", "lt":
		want, werr := e.resolveNumericArg(callArgs)
		if werr != nil {
			return werr
		}
		got, ok := toNumber(actual)
		if !ok {
			return fmt.Errorf("expected number for .%s, got %v", callName, actual)
		}
		passed = got < want
		failMsg = fmt.Sprintf("expected %v to be below %v", got, want)

	case "above", "greaterThan", "gt":
		want, werr := e.resolveNumericArg(callArgs)
		if werr != nil {
			return werr
		}
		got, ok := toNumber(actual)
		if !ok {
			return fmt.Errorf("expected number for .%s, got %v", callName, actual)
		}
		passed = got > want
		failMsg = fmt.Sprintf("expected %v to be above %v", got, want)

	case "least", "gte":
		want, werr := e.resolveNumericArg(callArgs)
		if werr != nil {
			return werr
		}
		got, ok := toNumber(actual)
		if !ok {
			return fmt.Errorf("expected number for .%s, got %v", callName, actual)
		}
		passed = got >= want
		failMsg = fmt.Sprintf("expected %v to be at least %v", got, want)

	case "most", "lte":
		want, werr := e.resolveNumericArg(callArgs)
		if werr != nil {
			return werr
		}
		got, ok := toNumber(actual)
		if !ok {
			return fmt.Errorf("expected number for .%s, got %v", callName, actual)
		}
		passed = got <= want
		failMsg = fmt.Sprintf("expected %v to be at most %v", got, want)

	case "include", "contain", "includes", "contains":
		if len(callArgs) < 1 {
			return fmt.Errorf("%s requires an argument", callName)
		}
		want, werr := e.resolveExpr(callArgs[0])
		if werr != nil {
			return werr
		}
		passed = valueIncludes(actual, want)
		failMsg = fmt.Sprintf("expected %v to include %v", actual, want)

	case "match":
		if len(callArgs) < 1 {
			return fmt.Errorf("match requires regex pattern")
		}
		pat := strings.TrimSpace(callArgs[0])
		if strings.HasPrefix(pat, "/") && strings.LastIndex(pat, "/") > 0 {
			last := strings.LastIndex(pat, "/")
			pat = pat[1:last]
		} else if v, verr := e.resolveExpr(pat); verr == nil {
			pat = fmt.Sprint(v)
		}
		re, rerr := regexp.Compile(pat)
		if rerr != nil {
			return fmt.Errorf("invalid regex %q: %w", pat, rerr)
		}
		passed = re.MatchString(fmt.Sprint(actual))
		failMsg = fmt.Sprintf("expected %v to match /%s/", actual, pat)

	case "a", "an":
		if len(callArgs) < 1 {
			return fmt.Errorf(".be.%s requires type name", callName)
		}
		want, werr := e.resolveExpr(callArgs[0])
		if werr != nil {
			return werr
		}
		wantType := strings.ToLower(fmt.Sprint(want))
		gotType := jsTypeOf(actual)
		passed = gotType == wantType
		failMsg = fmt.Sprintf("expected %s type %q, got %q", targetExpr, wantType, gotType)

	case "property":
		if len(callArgs) < 1 {
			return fmt.Errorf(".have.property requires key name")
		}
		keyVal, kerr := e.resolveExpr(callArgs[0])
		if kerr != nil {
			return kerr
		}
		key := fmt.Sprint(keyVal)
		m, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object for property lookup %q, got %T", key, actual)
		}
		propVal, exists := m[key]
		passed = exists
		failMsg = fmt.Sprintf("expected object to have property %q", key)
		if exists && len(callArgs) > 1 {
			wantVal, werr := e.resolveExpr(callArgs[1])
			if werr != nil {
				return werr
			}
			passed = valuesEqual(propVal, wantVal)
			failMsg = fmt.Sprintf("expected property %q value %v to equal %v", key, propVal, wantVal)
		}

	case "lengthOf", "length":
		want, werr := e.resolveNumericArg(callArgs)
		if werr != nil {
			return werr
		}
		gotLen, ok := valueLength(actual)
		if !ok {
			return fmt.Errorf("cannot measure length of %T", actual)
		}
		passed = float64(gotLen) == want
		failMsg = fmt.Sprintf("expected length %d to equal %.0f", gotLen, want)

	case "true":
		b, ok := actual.(bool)
		passed = ok && b
		failMsg = fmt.Sprintf("expected %v to be true", actual)

	case "false":
		b, ok := actual.(bool)
		passed = ok && !b
		failMsg = fmt.Sprintf("expected %v to be false", actual)

	case "null":
		passed = actual == nil
		failMsg = fmt.Sprintf("expected %v to be null", actual)

	case "undefined":
		passed = actual == nil
		failMsg = fmt.Sprintf("expected %v to be undefined", actual)

	default:
		return fmt.Errorf("unsupported expect matcher %q", callName)
	}

	if negated {
		if passed {
			return fmt.Errorf("negated expectation failed: %s", failMsg)
		}
		return nil
	}
	if !passed {
		return fmt.Errorf("%s", failMsg)
	}
	return nil
}

func (e *scriptEnv) resolveNumericArg(args []string) (float64, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("expected numeric argument")
	}
	v, err := e.resolveExpr(args[0])
	if err != nil {
		return 0, err
	}
	n, ok := toNumber(v)
	if !ok {
		return 0, fmt.Errorf("expected numeric value, got %v", v)
	}
	return n, nil
}

func (e *scriptEnv) evalBooleanExpr(expr string) (bool, error) {
	for _, op := range []string{"===", "!==", "==", "!=", ">=", "<=", ">", "<"} {
		idx := findTopLevelOperator(expr, op)
		if idx <= 0 {
			continue
		}
		lhsStr := strings.TrimSpace(expr[:idx])
		rhsStr := strings.TrimSpace(expr[idx+len(op):])
		lhs, err := e.resolveExpr(lhsStr)
		if err != nil {
			return false, err
		}
		rhs, err := e.resolveExpr(rhsStr)
		if err != nil {
			return false, err
		}
		switch op {
		case "===", "==":
			return valuesEqual(lhs, rhs), nil
		case "!==", "!=":
			return !valuesEqual(lhs, rhs), nil
		case ">", ">=", "<", "<=":
			ln, lok := toNumber(lhs)
			rn, rok := toNumber(rhs)
			if !lok || !rok {
				return false, fmt.Errorf("non-numeric operands for %s", op)
			}
			switch op {
			case ">":
				return ln > rn, nil
			case ">=":
				return ln >= rn, nil
			case "<":
				return ln < rn, nil
			case "<=":
				return ln <= rn, nil
			}
		}
	}

	val, err := e.resolveExpr(expr)
	if err != nil {
		return false, err
	}
	return isTruthy(val), nil
}

func (e *scriptEnv) resolveExpr(expr string) (any, error) {
	s := strings.TrimSpace(expr)
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}

	if unquoted, ok := parseJSStringLiteral(s); ok {
		return unquoted, nil
	}
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "undefined":
		return nil, nil
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n, nil
	}

	if s == "pm.response.code" || s == "pm.response.status" {
		return float64(e.resp.StatusCode), nil
	}
	if s == "pm.response.responseTime" {
		ms := e.resp.Duration.Milliseconds()
		if ms == 0 && e.resp.Telemetry.TotalMicros > 0 {
			return float64(e.resp.Telemetry.TotalMicros) / 1000.0, nil
		}
		return float64(ms), nil
	}
	if s == "pm.response.text()" {
		return e.resp.Body, nil
	}
	if s == "pm.response.size()" {
		sz := e.resp.SizeBytes
		if sz == 0 {
			sz = int64(len(e.resp.Body))
		}
		return float64(sz), nil
	}
	if strings.HasPrefix(s, "pm.response.headers.get(") && strings.HasSuffix(s, ")") {
		arg := s[len("pm.response.headers.get(") : len(s)-1]
		val, err := e.resolveExpr(arg)
		if err != nil {
			return nil, err
		}
		hv, ok := lookupHeaderCaseInsensitive(e.resp.Headers, fmt.Sprint(val))
		if !ok {
			return nil, nil
		}
		return hv, nil
	}
	if strings.HasPrefix(s, "pm.response.headers") {
		segs, err := parsePathSegments(strings.TrimPrefix(s, "pm.response.headers"))
		if err == nil && len(segs) == 1 {
			hv, ok := lookupHeaderCaseInsensitive(e.resp.Headers, segs[0].key)
			if !ok {
				return nil, nil
			}
			return hv, nil
		}
	}
	if strings.HasPrefix(s, "pm.response.json()") {
		root, err := e.parsedJSON()
		if err != nil {
			return nil, err
		}
		tail := strings.TrimPrefix(s, "pm.response.json()")
		return navigateValue(root, tail)
	}

	// Check bound script variables (e.g., data.user.id or json.length)
	segs, err := parsePathSegments(s)
	if err == nil && len(segs) > 0 && !segs[0].isIndex {
		if baseVal, exists := e.vars[segs[0].key]; exists {
			if storedErr, isErr := baseVal.(error); isErr {
				return nil, storedErr
			}
			return navigateSegments(baseVal, segs[1:])
		}
	}

	return nil, fmt.Errorf("unrecognized expression: %s", s)
}

type pathSegment struct {
	key     string
	index   int
	isIndex bool
}

func parsePathSegments(path string) ([]pathSegment, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	var segs []pathSegment
	i := 0
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
		case '[':
			closeIdx := strings.IndexByte(path[i:], ']')
			if closeIdx == -1 {
				return nil, fmt.Errorf("unterminated bracket in path %q", path)
			}
			inner := strings.TrimSpace(path[i+1 : i+closeIdx])
			if unq, ok := parseJSStringLiteral(inner); ok {
				segs = append(segs, pathSegment{key: unq})
			} else if idx, err := strconv.Atoi(inner); err == nil {
				segs = append(segs, pathSegment{index: idx, isIndex: true})
			} else {
				return nil, fmt.Errorf("unsupported index %q", inner)
			}
			i += closeIdx + 1
		default:
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			segs = append(segs, pathSegment{key: path[start:i]})
		}
	}
	return segs, nil
}

func navigateValue(root any, tail string) (any, error) {
	if strings.TrimSpace(tail) == "" {
		return root, nil
	}
	segs, err := parsePathSegments(tail)
	if err != nil {
		return nil, err
	}
	return navigateSegments(root, segs)
}

func navigateSegments(cur any, segs []pathSegment) (any, error) {
	for _, seg := range segs {
		if seg.isIndex {
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array at [%d], got %T", seg.index, cur)
			}
			if seg.index < 0 || seg.index >= len(arr) {
				return nil, fmt.Errorf("array index [%d] out of range (len=%d)", seg.index, len(arr))
			}
			cur = arr[seg.index]
			continue
		}
		if seg.key == "length" {
			if l, ok := valueLength(cur); ok {
				cur = float64(l)
				continue
			}
		}
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot read property %q of %T", seg.key, cur)
		}
		val, exists := obj[seg.key]
		if !exists {
			return nil, fmt.Errorf("property %q not found", seg.key)
		}
		cur = val
	}
	return cur, nil
}

func parsePMTestCall(stmt string) (title string, body string, ok bool) {
	prefix := ""
	switch {
	case strings.HasPrefix(stmt, "pm.test("):
		prefix = "pm.test("
	case strings.HasPrefix(stmt, "test("):
		prefix = "test("
	default:
		return "", "", false
	}

	openIdx := len(prefix) - 1
	closeIdx := matchingParen(stmt, openIdx)
	if closeIdx == -1 {
		return "", "", false
	}
	args := splitCallArgs(stmt[openIdx+1 : closeIdx])
	if len(args) < 2 {
		return "", "", false
	}
	tStr, tok := parseJSStringLiteral(strings.TrimSpace(args[0]))
	if !tok {
		tStr = strings.TrimSpace(args[0])
	}
	fn := strings.TrimSpace(args[1])
	braceStart := strings.IndexByte(fn, '{')
	braceEnd := strings.LastIndexByte(fn, '}')
	if braceStart == -1 || braceEnd <= braceStart {
		return "", "", false
	}
	return tStr, fn[braceStart+1 : braceEnd], true
}

func parseExpectChain(rest string) (modifiers []string, matcher string, args []string, err error) {
	if rest == "" {
		return nil, "", nil, fmt.Errorf("incomplete pm.expect chain")
	}
	// Separate trailing "(args)" if present at top level.
	rawChain := rest
	if openIdx := strings.IndexByte(rest, '('); openIdx != -1 {
		closeIdx := matchingParen(rest, openIdx)
		if closeIdx == -1 {
			return nil, "", nil, fmt.Errorf("unterminated matcher arguments in %q", rest)
		}
		rawChain = rest[:openIdx]
		args = splitCallArgs(rest[openIdx+1 : closeIdx])
	}

	parts := strings.Split(strings.TrimSuffix(rawChain, "."), ".")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return nil, "", nil, fmt.Errorf("missing matcher in %q", rest)
	}
	return parts[:len(parts)-1], parts[len(parts)-1], args, nil
}

func stripJSComments(src string) string {
	var out strings.Builder
	var inStr byte
	escaped := false
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if inStr != 0 {
			out.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == inStr {
				inStr = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			inStr = ch
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(src) {
			if src[i+1] == '/' {
				for i < len(src) && src[i] != '\n' {
					i++
				}
				if i < len(src) {
					out.WriteByte('\n')
				}
				continue
			}
			if src[i+1] == '*' {
				i += 2
				for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
					i++
				}
				i++
				continue
			}
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func splitTopLevelStatements(src string) []string {
	var stmts []string
	var buf strings.Builder
	var parenDepth, braceDepth, bracketDepth int
	var inStr byte
	escaped := false

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			stmts = append(stmts, s)
		}
		buf.Reset()
	}

	for i := 0; i < len(src); i++ {
		ch := src[i]
		if inStr != 0 {
			buf.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == inStr {
				inStr = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inStr = ch
			buf.WriteByte(ch)
		case '(':
			parenDepth++
			buf.WriteByte(ch)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			buf.WriteByte(ch)
		case '{':
			braceDepth++
			buf.WriteByte(ch)
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			buf.WriteByte(ch)
		case '[':
			bracketDepth++
			buf.WriteByte(ch)
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			buf.WriteByte(ch)
		case ';':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				flush()
			} else {
				buf.WriteByte(ch)
			}
		case '\n':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				flush()
			} else {
				buf.WriteByte(ch)
			}
		default:
			buf.WriteByte(ch)
		}
	}
	flush()
	return stmts
}

func splitCallArgs(inner string) []string {
	var args []string
	var buf strings.Builder
	var parenDepth, braceDepth, bracketDepth int
	var inStr byte
	escaped := false

	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if inStr != 0 {
			buf.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == inStr {
				inStr = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inStr = ch
			buf.WriteByte(ch)
		case '(':
			parenDepth++
			buf.WriteByte(ch)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			buf.WriteByte(ch)
		case '{':
			braceDepth++
			buf.WriteByte(ch)
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			buf.WriteByte(ch)
		case '[':
			bracketDepth++
			buf.WriteByte(ch)
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			buf.WriteByte(ch)
		case ',':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				args = append(args, strings.TrimSpace(buf.String()))
				buf.Reset()
				continue
			}
			buf.WriteByte(ch)
		default:
			buf.WriteByte(ch)
		}
	}
	if last := strings.TrimSpace(buf.String()); last != "" {
		args = append(args, last)
	}
	return args
}

func matchingParen(s string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(s) {
		return -1
	}
	depth := 0
	var inStr byte
	escaped := false
	for i := openIdx; i < len(s); i++ {
		ch := s[i]
		if inStr != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == inStr {
				inStr = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inStr = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findTopLevelOperator(s, op string) int {
	var parenDepth, braceDepth, bracketDepth int
	var inStr byte
	escaped := false
	for i := 0; i <= len(s)-len(op); i++ {
		ch := s[i]
		if inStr != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == inStr {
				inStr = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inStr = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		default:
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && strings.HasPrefix(s[i:], op) {
				return i
			}
		}
	}
	return -1
}

func parseJSStringLiteral(s string) (string, bool) {
	if len(s) < 2 {
		return "", false
	}
	q := s[0]
	if (q != '"' && q != '\'' && q != '`') || s[len(s)-1] != q {
		return "", false
	}
	return s[1 : len(s)-1], true
}

func lookupHeaderCaseInsensitive(headers map[string]string, name string) (string, bool) {
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

func toNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func valuesEqual(a, b any) bool {
	if an, aok := toNumber(a); aok {
		if bn, bok := toNumber(b); bok {
			return math.Abs(an-bn) < 1e-9
		}
	}
	return reflect.DeepEqual(a, b)
}

func valueIncludes(haystack, needle any) bool {
	if s, ok := haystack.(string); ok {
		return strings.Contains(s, fmt.Sprint(needle))
	}
	if arr, ok := haystack.([]any); ok {
		for _, item := range arr {
			if valuesEqual(item, needle) {
				return true
			}
		}
	}
	return false
}

func valueLength(v any) (int, bool) {
	switch x := v.(type) {
	case string:
		return len(x), true
	case []any:
		return len(x), true
	case map[string]any:
		return len(x), true
	default:
		return 0, false
	}
}

func jsTypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int64, float64, json.Number:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func isTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	default:
		return true
	}
}

func summarizeStmt(stmt string) string {
	if len(stmt) > 64 {
		return stmt[:61] + "..."
	}
	return stmt
}
