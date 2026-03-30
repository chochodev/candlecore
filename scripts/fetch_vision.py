import requests
import zipfile
import io
import pandas as pd
from datetime import datetime, timedelta
import os

def fetch_vision_data(symbol, interval, date_str):
    url = f"https://data.binance.vision/data/spot/daily/klines/{symbol}/{interval}/{symbol}-{interval}-{date_str}.zip"
    print(f"📥 Fetching {symbol} for {date_str}...")
    
    try:
        response = requests.get(url, timeout=15)
        if response.status_code != 200:
            print(f"⚠️  No data found for {date_str} (Status {response.status_code})")
            return None
        
        # Extract Zip
        with zipfile.ZipFile(io.BytesIO(response.content)) as z:
            csv_name = z.namelist()[0]
            with z.open(csv_name) as f:
                df = pd.read_csv(f, header=None)
                
        # Reformat: [timestamp, open, high, low, close, volume]
        # Vision Columns: 0:OpenTime, 1:Open, 2:High, 3:Low, 4:Close, 5:Volume
        df = df[[0, 1, 2, 3, 4, 5]]
        df.columns = ['timestamp', 'open', 'high', 'low', 'close', 'volume']
        
        # Convert timestamp (ms) to ISO
        df['timestamp'] = pd.to_datetime(df['timestamp'], unit='ms').dt.strftime('%Y-%m-%dT%H:%M:%SZ')
        
        return df
    except Exception as e:
        print(f"❌ Error: {e}")
        return None

def main():
    symbols = ['BTCUSDT', 'SOLUSDT', 'ETHUSDT']
    interval = '15m'
    
    # Try last 3 days to ensure we get a hit
    today = datetime.now()
    dates = [(today - timedelta(days=i)).strftime('%Y-%m-%d') for i in range(1, 4)]
    
    os.makedirs('data/historical', exist_ok=True)
    
    for sym in symbols:
        dfs = []
        for d in dates:
            df = fetch_vision_data(sym, interval, d)
            if df is not None:
                dfs.append(df)
        
        if dfs:
            final_df = pd.concat(dfs).sort_values('timestamp').drop_duplicates()
            short_name = sym.replace('USDT', '').lower()
            output_path = f'data/historical/{short_name}_15m.csv'
            final_df.to_csv(output_path, index=False)
            print(f"✅ Saved {len(final_df)} candles to {output_path}")
        else:
            print(f"🛑 Failed to fetch any data for {sym}")

if __name__ == "__main__":
    main()
