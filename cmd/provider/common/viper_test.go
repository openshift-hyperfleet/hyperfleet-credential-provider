package common

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearHFCPEnvVars saves and clears all HFCP_* environment variables
func clearHFCPEnvVars() map[string]string {
	savedEnvVars := map[string]string{}
	envVarsToClear := []string{
		"HFCP_LOG_LEVEL",
		"HFCP_LOG_FORMAT",
		"HFCP_CREDENTIALS_FILE",
		"HFCP_PROVIDER",
		"HFCP_CLUSTER_NAME",
		"HFCP_REGION",
		"HFCP_PROJECT_ID",
		"HFCP_ACCOUNT_ID",
		"HFCP_SUBSCRIPTION_ID",
		"HFCP_TENANT_ID",
		"HFCP_RESOURCE_GROUP",
		"HFCP_TOKEN_DURATION",
	}
	for _, key := range envVarsToClear {
		if val, exists := os.LookupEnv(key); exists {
			savedEnvVars[key] = val
			os.Unsetenv(key)
		}
	}
	return savedEnvVars
}

// restoreEnvVars restores previously saved environment variables
func restoreEnvVars(savedEnvVars map[string]string) {
	for key, val := range savedEnvVars {
		os.Setenv(key, val)
	}
}

func TestInitViper(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	// Reset viper before each test
	viper.Reset()

	InitViper()

	// Verify environment variable prefix is set
	os.Setenv("HFCP_TEST_KEY", "test-value")
	defer os.Unsetenv("HFCP_TEST_KEY")

	// Viper should automatically read the env var
	value := viper.GetString("test-key")
	assert.Equal(t, "test-value", value, "Viper should read environment variable with prefix")
}

func TestBindFlagsToViper_GlobalFlags(t *testing.T) {
	// Save and clear any existing HFCP_* environment variables
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	tests := []struct {
		name     string
		envVars  map[string]string
		initial  *Flags
		expected *Flags
	}{
		{
			name: "bind log-level from env",
			envVars: map[string]string{
				"HFCP_LOG_LEVEL": "debug",
			},
			initial: &Flags{
				LogLevel: "info", // default value
			},
			expected: &Flags{
				LogLevel: "debug", // should be overridden by env var
			},
		},
		{
			name: "bind log-format from env",
			envVars: map[string]string{
				"HFCP_LOG_FORMAT": "console",
			},
			initial: &Flags{
				LogFormat: "json",
			},
			expected: &Flags{
				LogFormat: "console",
			},
		},
		{
			name: "bind credentials-file from env",
			envVars: map[string]string{
				"HFCP_CREDENTIALS_FILE": "/path/to/creds.json",
			},
			initial: &Flags{
				CredentialsFile: "",
			},
			expected: &Flags{
				CredentialsFile: "/path/to/creds.json",
			},
		},
		{
			name: "multiple env vars",
			envVars: map[string]string{
				"HFCP_LOG_LEVEL":        "warn",
				"HFCP_LOG_FORMAT":       "console",
				"HFCP_CREDENTIALS_FILE": "/vault/secrets/sa.json",
			},
			initial: &Flags{
				LogLevel:        "info",
				LogFormat:       "json",
				CredentialsFile: "",
			},
			expected: &Flags{
				LogLevel:        "warn",
				LogFormat:       "console",
				CredentialsFile: "/vault/secrets/sa.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Re-initialize viper to pick up env vars
			viper.Reset()
			InitViper()

			// Create a test command
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("log-level", "", "log level")
			cmd.Flags().String("log-format", "", "log format")
			cmd.Flags().String("credentials-file", "", "credentials file")

			// Apply bindings
			flags := tt.initial
			BindFlagsToViper(cmd, flags)

			// Verify
			assert.Equal(t, tt.expected.LogLevel, flags.LogLevel)
			assert.Equal(t, tt.expected.LogFormat, flags.LogFormat)
			assert.Equal(t, tt.expected.CredentialsFile, flags.CredentialsFile)
		})
	}
}

func TestBindFlagsToViper_ProviderFlags(t *testing.T) {
	// Save and clear any existing HFCP_* environment variables
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	tests := []struct {
		name     string
		envVars  map[string]string
		initial  *Flags
		expected *Flags
	}{
		{
			name: "bind provider from env",
			envVars: map[string]string{
				"HFCP_PROVIDER": "gcp",
			},
			initial: &Flags{
				ProviderName: "",
			},
			expected: &Flags{
				ProviderName: "gcp",
			},
		},
		{
			name: "bind cluster-name from env",
			envVars: map[string]string{
				"HFCP_CLUSTER_NAME": "my-cluster",
			},
			initial: &Flags{
				ClusterName: "",
			},
			expected: &Flags{
				ClusterName: "my-cluster",
			},
		},
		{
			name: "bind all provider flags",
			envVars: map[string]string{
				"HFCP_PROVIDER":     "gcp",
				"HFCP_CLUSTER_NAME": "test-cluster",
				"HFCP_REGION":       "us-central1",
				"HFCP_PROJECT_ID":   "my-project",
			},
			initial: &Flags{},
			expected: &Flags{
				ProviderName: "gcp",
				ClusterName:  "test-cluster",
				Region:       "us-central1",
				ProjectID:    "my-project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Re-initialize viper
			viper.Reset()
			InitViper()

			// Create a test command
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("provider", "", "provider")
			cmd.Flags().String("cluster-name", "", "cluster name")
			cmd.Flags().String("region", "", "region")
			cmd.Flags().String("project-id", "", "project ID")

			// Apply bindings
			flags := tt.initial
			BindFlagsToViper(cmd, flags)

			// Verify
			assert.Equal(t, tt.expected.ProviderName, flags.ProviderName)
			assert.Equal(t, tt.expected.ClusterName, flags.ClusterName)
			assert.Equal(t, tt.expected.Region, flags.Region)
			assert.Equal(t, tt.expected.ProjectID, flags.ProjectID)
		})
	}
}

func TestBindFlagsToViper_AWSFlags(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	os.Setenv("HFCP_ACCOUNT_ID", "123456789012")
	defer os.Unsetenv("HFCP_ACCOUNT_ID")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("account-id", "", "account ID")

	flags := &Flags{}
	BindFlagsToViper(cmd, flags)

	assert.Equal(t, "123456789012", flags.AccountID)
}

func TestBindFlagsToViper_AzureFlags(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	envVars := map[string]string{
		"HFCP_SUBSCRIPTION_ID": "sub-123",
		"HFCP_TENANT_ID":       "tenant-456",
		"HFCP_RESOURCE_GROUP":  "my-rg",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("subscription-id", "", "subscription ID")
	cmd.Flags().String("tenant-id", "", "tenant ID")
	cmd.Flags().String("resource-group", "", "resource group")

	flags := &Flags{}
	BindFlagsToViper(cmd, flags)

	assert.Equal(t, "sub-123", flags.SubscriptionID)
	assert.Equal(t, "tenant-456", flags.TenantID)
	assert.Equal(t, "my-rg", flags.ResourceGroup)
}

func TestBindFlagsToViper_NoEnvVars(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-level", "", "log level")
	cmd.Flags().String("log-format", "", "log format")
	cmd.Flags().String("provider", "", "provider")

	// No environment variables set
	flags := &Flags{
		LogLevel:     "info",
		LogFormat:    "json",
		ProviderName: "",
	}

	BindFlagsToViper(cmd, flags)

	// When no env vars are set and no flags are set, viper returns empty strings
	// Since flags are not set explicitly, env var values are used (which are empty)
	assert.Equal(t, "", flags.LogLevel, "Viper returns empty string when no env var set")
	assert.Equal(t, "", flags.LogFormat, "Viper returns empty string when no env var set")
	assert.Equal(t, "", flags.ProviderName)
}

func TestBindFlagsToViper_TokenDuration(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	os.Setenv("HFCP_TOKEN_DURATION", "2h")
	defer os.Unsetenv("HFCP_TOKEN_DURATION")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("token-duration", "", "token duration")

	flags := &Flags{}
	BindFlagsToViper(cmd, flags)

	assert.Equal(t, "2h", flags.TokenDuration)
}

func TestBindCommandFlags(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Create a test command with flags
	cmd := &cobra.Command{
		Use: "test",
	}

	var testFlag string
	cmd.Flags().StringVar(&testFlag, "test-flag", "", "test flag")

	// Bind flags
	err := BindCommandFlags(cmd)
	require.NoError(t, err)

	// Set env var
	os.Setenv("HFCP_TEST_FLAG", "test-value")
	defer os.Unsetenv("HFCP_TEST_FLAG")

	// Viper should be able to read it
	value := viper.GetString("test-flag")
	assert.Equal(t, "test-value", value)
}

func TestBindPersistentFlags(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Create a test command with persistent flags
	rootCmd := &cobra.Command{
		Use: "root",
	}

	var testFlag string
	rootCmd.PersistentFlags().StringVar(&testFlag, "persistent-flag", "", "persistent test flag")

	// Bind persistent flags
	BindPersistentFlags(rootCmd)

	// Set env var
	os.Setenv("HFCP_PERSISTENT_FLAG", "persistent-value")
	defer os.Unsetenv("HFCP_PERSISTENT_FLAG")

	// Viper should be able to read it
	value := viper.GetString("persistent-flag")
	assert.Equal(t, "persistent-value", value)
}

func TestBindFlagsToViper_UnderscoreReplacement(t *testing.T) {
	// Test that hyphens in flag names are converted to underscores in env vars
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Set env var with underscores
	os.Setenv("HFCP_CLUSTER_NAME", "test-cluster")
	defer os.Unsetenv("HFCP_CLUSTER_NAME")

	viper.Reset()
	InitViper()

	// Viper should read it with hyphen key name
	value := viper.GetString("cluster-name")
	assert.Equal(t, "test-cluster", value, "Viper should convert underscores to hyphens")
}

func TestBindFlagsToViper_EmptyEnvVar(t *testing.T) {
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Set empty env var
	os.Setenv("HFCP_PROVIDER", "")
	defer os.Unsetenv("HFCP_PROVIDER")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("provider", "", "provider")

	flags := &Flags{
		ProviderName: "default-provider",
	}

	BindFlagsToViper(cmd, flags)

	// Empty env var should not override default when flag is not set
	// Note: This depends on implementation - empty string is still a valid value
	value := viper.GetString("provider")
	assert.Equal(t, "", value, "Empty env var is a valid value")
}

func TestBindFlagsToViper_FlagTakesPrecedence(t *testing.T) {
	// Test that explicitly set flags take precedence over environment variables
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Set environment variable
	os.Setenv("HFCP_PROVIDER", "from-env")
	os.Setenv("HFCP_CREDENTIALS_FILE", "/env/path/creds.json")
	defer os.Unsetenv("HFCP_PROVIDER")
	defer os.Unsetenv("HFCP_CREDENTIALS_FILE")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("provider", "", "provider")
	cmd.Flags().String("credentials-file", "", "credentials file")

	// Simulate user setting flags
	err := cmd.Flags().Set("provider", "from-flag")
	require.NoError(t, err)
	err = cmd.Flags().Set("credentials-file", "/flag/path/creds.json")
	require.NoError(t, err)

	// Bind flags to viper
	err = BindCommandFlags(cmd)
	require.NoError(t, err)

	flags := &Flags{
		ProviderName:    "from-flag",             // Set by flag
		CredentialsFile: "/flag/path/creds.json", // Set by flag
	}

	BindFlagsToViper(cmd, flags)

	// Flags should take precedence over environment variables
	assert.Equal(t, "from-flag", flags.ProviderName, "Flag should take precedence over env var")
	assert.Equal(t, "/flag/path/creds.json", flags.CredentialsFile, "Flag should take precedence over env var")
}

func TestBindFlagsToViper_EnvVarUsedWhenFlagNotSet(t *testing.T) {
	// Test that environment variables are used when flags are not set
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Set environment variable
	os.Setenv("HFCP_PROVIDER", "from-env")
	os.Setenv("HFCP_REGION", "us-west1")
	defer os.Unsetenv("HFCP_PROVIDER")
	defer os.Unsetenv("HFCP_REGION")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("provider", "", "provider")
	cmd.Flags().String("region", "", "region")

	// Bind flags to viper
	err := BindCommandFlags(cmd)
	require.NoError(t, err)

	// Don't set the flags - they should come from env vars
	flags := &Flags{}

	BindFlagsToViper(cmd, flags)

	// Environment variables should be used when flags are not set
	assert.Equal(t, "from-env", flags.ProviderName, "Env var should be used when flag not set")
	assert.Equal(t, "us-west1", flags.Region, "Env var should be used when flag not set")
}

func TestBindFlagsToViper_MixedFlagAndEnv(t *testing.T) {
	// Test mixed scenario: some flags set, some from env
	savedEnvVars := clearHFCPEnvVars()
	defer restoreEnvVars(savedEnvVars)

	viper.Reset()
	InitViper()

	// Set environment variables
	os.Setenv("HFCP_PROVIDER", "from-env")
	os.Setenv("HFCP_REGION", "from-env-region")
	defer os.Unsetenv("HFCP_PROVIDER")
	defer os.Unsetenv("HFCP_REGION")

	viper.Reset()
	InitViper()

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("provider", "", "provider")
	cmd.Flags().String("region", "", "region")

	// Only set provider flag, not region
	err := cmd.Flags().Set("provider", "from-flag")
	require.NoError(t, err)

	// Bind flags to viper
	err = BindCommandFlags(cmd)
	require.NoError(t, err)

	flags := &Flags{
		ProviderName: "from-flag", // Set by flag
	}

	BindFlagsToViper(cmd, flags)

	// Provider should use flag value, region should use env var
	assert.Equal(t, "from-flag", flags.ProviderName, "Flag should take precedence")
	assert.Equal(t, "from-env-region", flags.Region, "Env var should be used when flag not set")
}
