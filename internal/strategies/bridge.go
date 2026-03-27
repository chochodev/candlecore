package strategies

import (
	"candlecore/internal/bot"
	"candlecore/internal/strategy"
)

// GetStrategy is a bridge to the strategy registry, returns adapted strategy
func GetStrategy(name string) (bot.Strategy, error) {
	s, err := strategy.Get(name)
	if err != nil {
		return nil, err
	}
	return NewStrategyAdapter(s), nil
}

// ListStrategies is a bridge to list all available strategies
func ListStrategies() []string {
	return strategy.List()
}
