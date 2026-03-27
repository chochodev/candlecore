# Candlecore Production Upgrade Plan
**Transform Candlecore to Freqtrade-Level Quality in Go**

## Implementation Strategy

### Phase 1: Core Strategy System (PRIORITY 1)
**Goal:** Pluggable strategy architecture

#### Step 1.1: Strategy Interface
- [ ] Create `internal/strategy/interface.go`
  - Define `IStrategy` interface
  - Methods: `PopulateIndicators()`, `PopulateEntry()`, `PopulateExit()`
  - Config: `MinimalROI`, `Stoploss`, `Timeframe`, `TrailingStop`
  
#### Step 1.2: Strategy Registry
- [ ] Create `internal/strategy/registry.go`
  - Factory pattern for strategy creation
  - `RegisterStrategy()`, `GetStrategy()`, `ListStrategies()`

#### Step 1.3: Refactor Existing Strategies
- [ ] Move MA crossover to `internal/strategy/ma_crossover.go`
- [ ] Move RSI to `internal/strategy/rsi.go`
- [ ] Both implement `IStrategy` interface

**Test:** Bot can switch strategies without code changes

---

### Phase 2: Risk Management System (PRIORITY 1)
**Goal:** Professional risk controls

#### Step 2.1: Stop Loss Manager
- [ ] Create `internal/risk/stoploss.go`
  - Fixed stoploss (percentage-based)
  - Trailing stoploss (follows price)
  - Custom stoploss (strategy callback)
  
#### Step 2.2: ROI Manager
- [ ] Create `internal/risk/roi.go`
  - Time-based minimal ROI table
  - Example: `{0: 0.10, 60: 0.05, 120: 0.02}`
  - Check on each candle if ROI target hit

#### Step 2.3: Position Sizing
- [ ] Create `internal/risk/position_sizing.go`
  - Calculate trade size based on risk
  - Max position % of portfolio
  - Stake amount calculation

**Test:** Bot respects stoploss and ROI exits

---

### Phase 3: Enhanced Bot Engine (PRIORITY 1)
**Goal:** Smarter decision-making

#### Step 3.1: Decision Engine Refactor
- [ ] Update `internal/bot/decision.go`
  - Multi-factor signal scoring
  - Confidence calculation (0-100%)
  - Entry/Exit reasons tracking
  
#### Step 3.2: Bot Controller Update
- [ ] Refactor `internal/bot/controller.go`
  - Use strategy interface
  - Integrate risk management
  - Track exit reasons

#### Step 3.3: Trade State Machine
- [ ] Create `internal/bot/trade_state.go`
  - States: `PENDING`, `OPEN`, `CLOSING`, `CLOSED`
  - Track entry time, exit time, reason
  - P&L calculation improvements

**Test:** Bot makes better decisions with clear reasoning

---

### Phase 4: Dry-Run Mode (PRIORITY 2)
**Goal:** Safe testing without real capital

#### Step 4.1: Virtual Wallet
- [ ] Create `internal/exchange/virtual_wallet.go`
  - Simulated balance
  - Track virtual trades
  - Calculate virtual P&L

#### Step 4.2: Dry-Run Exchange
- [ ] Create `internal/exchange/dry_run.go`
  - Implements exchange interface
  - Simulates order execution
  - Uses real market data but fake orders

#### Step 4.3: Config Toggle
- [ ] Add `dry_run` boolean to config
- [ ] Bot selects real/virtual exchange based on mode

**Test:** Bot runs in dry-run, no real trades executed

---

### Phase 5: Data Provider Pattern (PRIORITY 2)
**Goal:** Centralized data access

#### Step 5.1: Data Provider
- [ ] Create `internal/data/provider.go`
  - `GetOHLCV(pair, timeframe)` - Get candles
  - `GetTicker(pair)` - Get current price
  - `GetOrderbook(pair)` - Get orderbook
  - `GetTrades()` - Get bot's trades

#### Step 5.2: Integrate with Strategy
- [ ] Strategies get data via provider
- [ ] Can access multiple timeframes
- [ ] Can access multiple pairs

**Test:** Strategies can access any market data needed

---

### Phase 6: Position Management (PRIORITY 3)
**Goal:** Advanced position handling

#### Step 6.1: Position Adjustment
- [ ] Create `internal/bot/position_adjustment.go`
  - DCA (Dollar Cost Averaging)
  - Partial exits
  - Position increase logic

#### Step 6.2: Pair Locking
- [ ] Create `internal/bot/pair_lock.go`
  - Lock pair for X minutes
  - Unlock automatically
  - Check before entry

**Test:** Bot can adjust positions and lock pairs

---

### Phase 7: API Enhancements (PRIORITY 2)
**Goal:** Full control via API

#### Step 7.1: Strategy Management Endpoints
- [ ] `GET /api/v1/strategies` - List available strategies
- [ ] `POST /api/v1/bot/strategy` - Switch strategy
- [ ] `GET /api/v1/strategy/config` - Get strategy config

#### Step 7.2: Risk Management Endpoints
- [ ] `GET /api/v1/risk/stoploss` - Get stoploss config
- [ ] `POST /api/v1/risk/stoploss` - Update stoploss
- [ ] `GET /api/v1/risk/roi` - Get ROI table

#### Step 7.3: Dry-Run Toggle
- [ ] `POST /api/v1/bot/mode` - Switch dry/live mode
- [ ] `GET /api/v1/wallet` - Get balance (real or virtual)

#### Step 7.4: Trade History
- [ ] `GET /api/v1/trades` - All trades with filters
- [ ] `GET /api/v1/trade/{id}` - Single trade details
- [ ] Include entry/exit reasons

**Test:** All new features accessible via API

---

### Phase 8: Frontend Updates (PRIORITY 3)
**Goal:** Display new features

#### Step 8.1: Strategy Selector
- [ ] Add strategy dropdown in config modal
- [ ] Show strategy parameters
- [ ] Display active strategy

#### Step 8.2: Risk Indicators
- [ ] Show stoploss level on chart
- [ ] Show ROI targets
- [ ] Display trailing stop status

#### Step 8.3: Trade Details
- [ ] Show entry/exit reasons
- [ ] Display confidence scores
- [ ] Dry-run mode indicator

**Test:** Frontend shows all new bot capabilities

---

## Success Criteria

✅ **Strategy System:**
- Can add new strategies without touching core bot code
- Strategies are self-contained and reusable
- Easy to switch between strategies

✅ **Risk Management:**
- Bot never exceeds risk limits
- Stop losses always execute
- ROI targets enforced

✅ **Production Ready:**
- Dry-run mode works perfectly
- No capital risk during testing
- Clear logging and error handling

✅ **User Experience:**
- Easy to configure via API
- Clear feedback on decisions
- Transparent reasoning

---

## Implementation Order

1. **Phase 1** → Strategy system (foundation)
2. **Phase 2** → Risk management (safety)
3. **Phase 3** → Bot engine improvements (intelligence)
4. **Phase 7** → API updates (as we go)
5. **Phase 4** → Dry-run mode (testing)
6. **Phase 5** → Data provider (optional enhancement)
7. **Phase 6** → Position management (advanced)
8. **Phase 8** → Frontend polish (last)

---

## Development Approach

**Incremental & Safe:**
1. Build one phase at a time
2. Test after each step
3. Keep bot functional throughout
4. No breaking changes to existing API
5. Add features, don't replace

**Quality Standards:**
- Go best practices
- Comprehensive error handling
- Thread-safe operations
- Clear logging
- Type safety
