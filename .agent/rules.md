# Candlecore Project Rules

## Market Scope

### Target Markets

- Crypto spot markets only (BTC/USDT, ETH/USDT)
- No futures, options, or leveraged instruments
- No stock or forex markets
- Medium timeframes: 5m, 15m candles
- Paper trading and backtesting only

### Data Sources

- Free public APIs (Binance)
- Live 24/7 market data
- No paid data feeds
- No authentication required for public endpoints

### Constraints

- Candle-based strategies only
- No real money trading
- No derivatives or leverage
- No multi-asset portfolio logic (single instrument focus)

### Out of Scope

- Real capital deployment
- Live exchange integration for order execution
- Paid market data
- Stock/forex markets
- High-frequency trading

## Documentation Standards

- Do not create markdown files to document changes
- Do not create summary files, changelog files, or completion reports
- Code comments should be sufficient documentation
- Only update existing documentation files when explicitly requested

## Code Style

- No emojis in code, comments, logs, or any output
- Use plain text for all messages and documentation
- Professional, technical communication style only

## Development Standards

- This is a production-ready trading bot
- Never use placeholders, TODOs, or "implement later" comments
- All code must be complete and robust on first implementation
- No stub functions or temporary implementations
- If a feature cannot be fully implemented, discuss alternatives rather than leaving it incomplete
- All error handling must be comprehensive and production-grade
- All configurations must have sensible defaults
- All integrations must be fully functional

## Testing and Quality

- All code must be production-ready
- Proper error handling is mandatory
- No debug-only code paths
- All features must be fully implemented and tested
- Configuration validation must be thorough

## Implementation Approach

- When adding features, implement them completely
- Include all necessary error handling, logging, and edge cases
- No "// TODO: implement this later" comments
- If uncertain about implementation, ask for clarification before proceeding
- Default to robust, complete implementations over quick prototypes

## Communication Style

- Go straight to the point
- Be concise and direct in explanations
- Avoid overwhelming with excessive content or unnecessary detail
- Answer what was asked without over-explaining

## Development Documentation

- Update dev.md when adding new CLI commands
- Document critical development decisions in dev.md
- Add troubleshooting steps for common issues
- Keep command examples up to date
- Document new dependencies and their purposes
