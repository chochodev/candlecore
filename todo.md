# Solana Research & Strategy Pipeline

## Phase 1: Data Infrastructure & Extraction (The Foundation)
- [ ] **Step 1 - Decommission Legacy Strategy**: Stop all ad-hoc logic/bot-tweaking to prevent strategy bias.
- [x] **Step 2 - Multiscale Data Transformer**: Build `internal/research` to align M15/H1.
- [x] **Step 3 - Feature Engineering Engine**: Extract statistical price signatures.
- [x] **Step 4 - Triple Barrier Labeling**: Implement Prado-style volatility labeling.

## Phase 2: Statistical Edge Discovery
- [ ] **Step 5 - Dataset Construction**: Aggregate and export labeled datasets.
- [ ] **Step 6 - Condition Mining**: Groups clusters to find >55% win-rate edges.
- [x] **Step 7 - Expectancy Ranking**: Sort conditions by profit significance.

## Phase 3: Walk-Forward Validation (Reality Check)
- [ ] **Step 8 - 70/30 Train/Test Split**: Optimize on 70% / Validate on 30%.
- [ ] **Step 9 - Drift Rejection**: Reject conditions that collapse out-of-sample.
- [ ] **Step 10 - Friction Injection**: Simulate spread/slippage on winning sets.

## Phase 4: Strategy Codification & Deployment
- [ ] **Step 11 - Hardcoded Rule Generation**: Convert winners into code.
- [ ] **Step 12 - Performance Guarding**: Detect edge decay in real-time.