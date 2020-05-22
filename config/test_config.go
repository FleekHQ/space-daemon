package config

// TestConfig is an easily mockable config that can be used in tests
type TestConfig struct {
	config map[string]interface{}
}

func NewTestConfig(config map[string]interface{}) Config {
	return &TestConfig{
		config: config,
	}
}

func (c *TestConfig) GetString(key string, defaultValue interface{}) string {
	return c.config[key].(string)
}

func (c *TestConfig) GetInt(key string, defaultValue interface{}) int {
	return c.config[key].(int)
}
