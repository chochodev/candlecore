import yfinance as yf
import pandas as pd
import os
from datetime import datetime, timedelta

def main():
    symbols = {
        'SOL-USD': 'sol',
        'BTC-USD': 'btc',
        'ETH-USD': 'eth'
    }
    
    os.makedirs('data/historical', exist_ok=True)
    
    for yf_sym, short_name in symbols.items():
        print(f"📥 Fetching 60 Days of 5m data for {yf_sym}...")
        
        # Yahoo 5m data is available for the last 60 days
        data = yf.download(yf_sym, interval="5m", period="60d")
        
        if not data.empty:
            if isinstance(data.columns, pd.MultiIndex):
                data.columns = data.columns.get_level_values(0)
            
            df = data.reset_index()
            # Rename based on actual columns returned
            df = df[['Datetime', 'Open', 'High', 'Low', 'Close', 'Volume']]
            df.columns = ['timestamp', 'open', 'high', 'low', 'close', 'volume']
            df['timestamp'] = df['timestamp'].dt.strftime('%Y-%m-%dT%H:%M:%SZ')
            
            output_path = f'data/historical/{short_name}_5m.csv'
            df.to_csv(output_path, index=False)
            print(f"✅ Saved {len(df)} candles to {output_path}")

if __name__ == "__main__":
    main()
