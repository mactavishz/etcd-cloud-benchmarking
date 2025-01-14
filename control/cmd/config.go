package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"time"
	"unicode"

	benchCfg "csb/control/config"
	constants "csb/control/constants"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage benchctl configuration",
	Long:  "View and modify benchctl configuration settings",
}

var configSetCmd = &cobra.Command{
	Use:   "set field=value",
	Short: "Set a configuration field",
	Long:  "Set the value of a specific configuration field (e.g., config set seed=12345)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		parts := strings.Split(args[0], "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format. Use: field=value")
		}
		field, value := parts[0], parts[1]

		// Convert snake_case to camelCase for field lookup
		field = toCamelCase(field)

		// Get reflection value of config struct
		configVal := reflect.ValueOf(GConfig.ctlConfig).Elem()
		fieldVal := configVal.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(s, field)
		})

		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", field)
		}

		// Handle time.Duration fields specially
		if fieldVal.Type() == reflect.TypeOf(benchCfg.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration value for %s: %w", field, err)
			}
			fieldVal.Set(reflect.ValueOf(benchCfg.Duration(duration)))

			// Save the updated configuration
			if err := benchCfg.ValidateConfig(GConfig.ctlConfig); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			return GConfig.ctlConfig.WriteConfig(GConfig.GetConfigFilePath())
		}

		// Convert and set the value based on field type
		switch fieldVal.Kind() {
		case reflect.Int64:
			var v int64
			_, err := fmt.Sscanf(value, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %w", field, err)
			}
			fieldVal.SetInt(v)
		case reflect.Int:
			var v int
			_, err := fmt.Sscanf(value, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %w", field, err)
			}
			fieldVal.SetInt(int64(v))
		case reflect.Float64:
			var v float64
			_, err := fmt.Sscanf(value, "%f", &v)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %w", field, err)
			}
			fieldVal.SetFloat(v)
		case reflect.String:
			fieldVal.SetString(value)
		case reflect.Slice:
			if fieldVal.Type().Elem().Kind() == reflect.String {
				values := strings.Split(value, ",")
				slice := reflect.MakeSlice(fieldVal.Type(), len(values), len(values))
				for i, v := range values {
					slice.Index(i).SetString(strings.TrimSpace(v))
				}
				fieldVal.Set(slice)
			} else {
				return fmt.Errorf("unsupported slice type for field %s", field)
			}
		default:
			return fmt.Errorf("unsupported type for field %s", field)
		}

		// Validate the new configuration
		if err := benchCfg.ValidateConfig(GConfig.ctlConfig); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		// Save the updated configuration
		return GConfig.ctlConfig.WriteConfig(GConfig.GetConfigFilePath())
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get field",
	Short: "Get a configuration field value",
	Long:  "Get the current value of a specific configuration field",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		field := args[0]

		field = toCamelCase(field)

		// Get reflection value of config struct
		configVal := reflect.ValueOf(GConfig.ctlConfig).Elem()
		fieldVal := configVal.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(s, field)
		})

		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", field)
		}

		// Print the field value
		fmt.Printf("%v\n", fieldVal.Interface())
		return nil
	},
}

var configLoadFileCmd = &cobra.Command{
	Use:   "load-file path/to/config.json",
	Short: "Load configuration from file",
	Long:  "Load and replace current configuration with contents from specified JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newConfig, err := benchCfg.ReadConfig(args[0])
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}

		// Update global config
		GConfig.ctlConfig = newConfig
		GConfig.UpdateRg(newConfig.Seed)

		err = initConfigDir()
		if err != nil {
			return err
		}

		// Save the new configuration
		return GConfig.ctlConfig.WriteConfig(GConfig.GetConfigFilePath())
	},
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  "View the current configuration in JSON format",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		data, err := json.MarshalIndent(GConfig.ctlConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration fields",
	Long:  "List all available configuration fields with their types and current values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		configVal := reflect.ValueOf(GConfig.ctlConfig).Elem()
		configType := configVal.Type()

		fmt.Printf("%-20s %-15s %-15s %s\n", "FIELD", "TYPE", "REQUIRED", "CURRENT VALUE")
		fmt.Println(strings.Repeat("-", 80))

		for i := 0; i < configVal.NumField(); i++ {
			fieldVal := configVal.Field(i)
			fieldType := configType.Field(i)

			// Get validation tags
			validateTag := fieldType.Tag.Get("validate")
			required := strings.Contains(validateTag, "required")

			// Format the type string
			typeStr := fieldType.Type.String()
			if fieldVal.Kind() == reflect.Slice {
				typeStr = fmt.Sprintf("[]%s", fieldType.Type.Elem().String())
			}

			// Format the current value
			var valueStr string
			if fieldVal.Kind() == reflect.Slice {
				sliceVals := make([]string, fieldVal.Len())
				for j := 0; j < fieldVal.Len(); j++ {
					sliceVals[j] = fmt.Sprint(fieldVal.Index(j).Interface())
				}
				if len(sliceVals) == 0 {
					valueStr = "[]"
				} else {
					valueStr = strings.Join(sliceVals, ",")
				}
			} else {
				valueStr = fmt.Sprint(fieldVal.Interface())
			}

			// Convert field name to snake_case
			fieldName := fieldType.Name
			snakeCase := toSnakeCase(fieldName)

			fmt.Printf("%-20s %-15s %-15v %s\n",
				snakeCase,
				typeStr,
				required,
				valueStr)
		}
		return nil
	},
}

// toSnakeCase converts a camelCase string to snake_case, handling acronyms properly
func toSnakeCase(s string) string {
	var result strings.Builder
	var prev rune
	for i, r := range s {
		if i > 0 {
			// Check if current char is uppercase and previous char is lowercase
			// or if current char is uppercase and next char is lowercase
			if unicode.IsUpper(r) {
				if unicode.IsLower(prev) ||
					(i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
					result.WriteRune('_')
				}
			}
		}
		result.WriteRune(unicode.ToLower(r))
		prev = r
	}
	return result.String()
}

// toCamelCase converts a snake_case string to camelCase, handling acronyms properly
func toCamelCase(s string) string {
	// Known acronyms that should be handled specially
	acronyms := map[string]bool{
		"sla": true,
	}

	parts := strings.Split(s, "_")
	var result strings.Builder

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Check if this part is a known acronym
		if acronyms[strings.ToLower(part)] {
			result.WriteString(strings.ToUpper(part))
			continue
		}

		// For normal words, capitalize first letter if not first word
		if i == 0 {
			result.WriteString(part)
		} else {
			result.WriteString(strings.ToUpper(part[0:1]) + part[1:])
		}
	}

	return result.String()
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Long:  "Initialize the configuration with default values and save it in JSON format in the config directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := initConfigDir()
		if err != nil {
			return err
		}
		return initConfigFile()
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset to default configuration",
	Long:  "Reset the configuration with default values and save it in JSON format in the config directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := initConfigDir()
		if err != nil {
			return err
		}
		return initConfigFile()
	},
}

func init() {
	ConfigCmd.AddCommand(configInitCmd)
	ConfigCmd.AddCommand(configResetCmd)
	ConfigCmd.AddCommand(configSetCmd)
	ConfigCmd.AddCommand(configGetCmd)
	ConfigCmd.AddCommand(configLoadFileCmd)
	ConfigCmd.AddCommand(configViewCmd)
	ConfigCmd.AddCommand(configListCmd)
}

func initConfigDir() error {
	if _, err := os.Stat(GConfig.ctlConfigPath); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(GConfig.ctlConfigPath, 0755); err != nil {
				fmt.Println("Failed to create config directory: ", err)
				return err
			}
		} else {
			fmt.Println("Failed to check config directory: ", err)
			return err
		}
	}
	return nil
}

func initConfigFile() error {
	configFilePath := path.Join(GConfig.ctlConfigPath, constants.DEFAULT_CONFIG_FILE)
	defaultConfig := benchCfg.GetDefaultConfig()

	GConfig.UpdateRg(defaultConfig.Seed)
	GConfig.ctlConfig = defaultConfig
	err := defaultConfig.WriteConfig(configFilePath)
	if err != nil {
		fmt.Println("Failed to write default config file: ", err)
		return err
	}
	fmt.Println("Default configuration initialized and saved in ", configFilePath)
	return nil
}
