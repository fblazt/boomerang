package env

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fblazt/boomerang/internal/models"
)

// Aliases for models types to make importing internal/engine/env ergonomic.
type Variable = models.Variable
type VariableType = models.VariableType
type Environment = models.Environment
type TransportPolicy = models.TransportPolicy

// Engine manages workspace environments, encrypted secrets, and variable resolution.
type Engine struct {
	mu           sync.RWMutex
	vault        *SecretVault
	environments []models.Environment
	activeEnv    string
}

// NewEngine initializes an EnvironmentEngine with an encrypted vault and default profiles.
func NewEngine(vault *SecretVault, defaultEnvs []models.Environment) *Engine {
	if len(defaultEnvs) == 0 {
		defaultEnvs = models.DefaultEnvironments()
	}
	active := "PROD"
	for _, e := range defaultEnvs {
		if e.IsDefault {
			active = e.Name
			break
		}
	}

	eng := &Engine{
		vault:        vault,
		environments: defaultEnvs,
		activeEnv:    active,
	}

	// Populate vault with initial secret variables if vault is provided
	if vault != nil {
		for _, e := range defaultEnvs {
			for _, v := range e.Variables {
				if v.Type == models.VariableTypeSecret && v.CurrentValue != "" {
					_ = vault.SetSecret(e.Name, v.Key, v.CurrentValue)
				}
			}
		}
	}

	return eng
}

// Vault returns the underlying SecretVault instance.
func (e *Engine) Vault() *SecretVault {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.vault
}

// GetEnvironments returns a copy of all configured workspace environments.
func (e *Engine) GetEnvironments() []models.Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]models.Environment, len(e.environments))
	copy(out, e.environments)
	return out
}

// GetEnvironment retrieves an environment by Name or ID.
func (e *Engine) GetEnvironment(nameOrID string) (*models.Environment, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, env := range e.environments {
		if strings.EqualFold(env.Name, nameOrID) || env.ID == nameOrID {
			cpy := env
			return &cpy, nil
		}
	}
	return nil, fmt.Errorf("environment %q not found", nameOrID)
}

// CreateEnvironment adds a new environment to the engine.
func (e *Engine) CreateEnvironment(env models.Environment) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, existing := range e.environments {
		if strings.EqualFold(existing.Name, env.Name) {
			return fmt.Errorf("environment %q already exists", env.Name)
		}
	}

	if env.ID == "" {
		env.ID = "env_" + strings.ToLower(env.Name)
	}
	e.environments = append(e.environments, env)
	return nil
}

// UpdateEnvironment replaces an existing environment's variables and settings.
func (e *Engine) UpdateEnvironment(env models.Environment) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, existing := range e.environments {
		if existing.ID == env.ID || strings.EqualFold(existing.Name, env.Name) {
			e.environments[i] = env
			return nil
		}
	}
	return fmt.Errorf("environment %q not found", env.Name)
}

// DeleteEnvironment removes an environment and its stored secrets.
func (e *Engine) DeleteEnvironment(nameOrID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	index := -1
	var envName string
	for i, existing := range e.environments {
		if existing.ID == nameOrID || strings.EqualFold(existing.Name, nameOrID) {
			index = i
			envName = existing.Name
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("environment %q not found", nameOrID)
	}

	if len(e.environments) <= 1 {
		return fmt.Errorf("cannot delete the only remaining environment")
	}

	e.environments = append(e.environments[:index], e.environments[index+1:]...)

	if e.activeEnv == envName {
		e.activeEnv = e.environments[0].Name
	}

	if e.vault != nil {
		secrets := e.vault.ListSecrets(envName)
		for _, k := range secrets {
			_ = e.vault.DeleteSecret(envName, k)
		}
	}

	return nil
}

// DuplicateEnvironment clones an environment under a new name.
func (e *Engine) DuplicateEnvironment(nameOrID string, newName string) (*models.Environment, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var source *models.Environment
	for _, env := range e.environments {
		if env.ID == nameOrID || strings.EqualFold(env.Name, nameOrID) {
			cpy := env
			source = &cpy
			break
		}
	}

	if source == nil {
		return nil, fmt.Errorf("source environment %q not found", nameOrID)
	}

	for _, env := range e.environments {
		if strings.EqualFold(env.Name, newName) {
			return nil, fmt.Errorf("environment %q already exists", newName)
		}
	}

	newEnv := *source
	newEnv.ID = "env_" + strings.ToLower(newName)
	newEnv.Name = newName
	newEnv.IsDefault = false

	// Clone variables
	newVars := make([]models.Variable, len(source.Variables))
	copy(newVars, source.Variables)
	newEnv.Variables = newVars

	e.environments = append(e.environments, newEnv)

	// Clone secrets in vault
	if e.vault != nil {
		secrets := e.vault.ListSecrets(source.Name)
		for _, k := range secrets {
			if secVal, err := e.vault.GetSecret(source.Name, k); err == nil {
				_ = e.vault.SetSecret(newName, k, secVal)
			}
		}
	}

	return &newEnv, nil
}

// SetActiveEnvironment switches the active workspace environment.
func (e *Engine) SetActiveEnvironment(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, env := range e.environments {
		if strings.EqualFold(env.Name, name) {
			e.activeEnv = env.Name
			return nil
		}
	}
	return fmt.Errorf("environment %q not found", name)
}

// GetActiveEnvironment returns a copy of the currently active Environment.
func (e *Engine) GetActiveEnvironment() *models.Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, env := range e.environments {
		if env.Name == e.activeEnv {
			cpy := env
			return &cpy
		}
	}
	if len(e.environments) > 0 {
		cpy := e.environments[0]
		return &cpy
	}
	return nil
}

// AddVariable adds a new variable to the specified environment.
func (e *Engine) AddVariable(envName string, v models.Variable, secretPlaintext string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.environments {
		if strings.EqualFold(e.environments[i].Name, envName) {
			for _, existing := range e.environments[i].Variables {
				if existing.Key == v.Key {
					return fmt.Errorf("variable %q already exists in %q", v.Key, envName)
				}
			}

			if v.Type == models.VariableTypeSecret && e.vault != nil {
				plaintext := secretPlaintext
				if plaintext == "" {
					plaintext = v.CurrentValue
				}
				_ = e.vault.SetSecret(envName, v.Key, plaintext)
				v.CurrentValue = MaskSecret(plaintext)
				v.InitialValue = MaskSecret(v.InitialValue)
			}

			e.environments[i].Variables = append(e.environments[i].Variables, v)
			return nil
		}
	}
	return fmt.Errorf("environment %q not found", envName)
}

// UpdateVariable updates an existing variable's values or enabled state.
func (e *Engine) UpdateVariable(envName string, v models.Variable, secretPlaintext string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.environments {
		if strings.EqualFold(e.environments[i].Name, envName) {
			for j := range e.environments[i].Variables {
				if e.environments[i].Variables[j].Key == v.Key {
					if v.Type == models.VariableTypeSecret && e.vault != nil {
						if secretPlaintext != "" {
							_ = e.vault.SetSecret(envName, v.Key, secretPlaintext)
							v.CurrentValue = MaskSecret(secretPlaintext)
						}
					}
					e.environments[i].Variables[j] = v
					return nil
				}
			}
			return fmt.Errorf("variable %q not found in %q", v.Key, envName)
		}
	}
	return fmt.Errorf("environment %q not found", envName)
}

// DeleteVariable removes a variable from an environment.
func (e *Engine) DeleteVariable(envName string, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.environments {
		if strings.EqualFold(e.environments[i].Name, envName) {
			vars := e.environments[i].Variables
			for j := range vars {
				if vars[j].Key == key {
					e.environments[i].Variables = append(vars[:j], vars[j+1:]...)
					if e.vault != nil {
						_ = e.vault.DeleteSecret(envName, key)
					}
					return nil
				}
			}
			return fmt.Errorf("variable %q not found in %q", key, envName)
		}
	}
	return fmt.Errorf("environment %q not found", envName)
}

// GetResolvedVariables builds the flat map of variable keys to their decrypted runtime values.
func (e *Engine) GetResolvedVariables(envName string) (map[string]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var target *models.Environment
	for _, env := range e.environments {
		if strings.EqualFold(env.Name, envName) {
			cpy := env
			target = &cpy
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("environment %q not found", envName)
	}

	resolved := make(map[string]string)
	for _, v := range target.Variables {
		if !v.Enabled {
			continue
		}

		if v.Type == models.VariableTypeSecret && e.vault != nil {
			if dec, err := e.vault.GetSecret(target.Name, v.Key); err == nil && dec != "" {
				resolved[v.Key] = dec
				continue
			}
		}

		val := v.CurrentValue
		if val == "" {
			val = v.InitialValue
		}
		resolved[v.Key] = val
	}

	return resolved, nil
}

// ExportEnvironment exports an environment to .env format.
func (e *Engine) ExportEnvironment(envName string, maskSecrets bool) (string, error) {
	env, err := e.GetEnvironment(envName)
	if err != nil {
		return "", err
	}
	return ExportDotEnv(*env, e.vault, maskSecrets), nil
}

// ImportEnvironment imports a .env format string into a new or updated environment.
func (e *Engine) ImportEnvironment(name string, dotEnvContent string) (*models.Environment, error) {
	env, secrets, err := ImportDotEnv(name, dotEnvContent)
	if err != nil {
		return nil, err
	}

	if err := e.CreateEnvironment(*env); err != nil {
		// If already exists, update
		_ = e.UpdateEnvironment(*env)
	}

	if e.vault != nil {
		for k, v := range secrets {
			_ = e.vault.SetSecret(name, k, v)
		}
	}

	return env, nil
}
