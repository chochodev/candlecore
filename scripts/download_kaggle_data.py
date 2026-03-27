#!/usr/bin/env python3
"""
Download Bitcoin Historical Data from Kaggle

This script downloads the comprehensive Bitcoin dataset from Kaggle:
- 1-minute interval data from 2012 to present
- Data from Bitstamp exchange
- Updated daily via automation

Dataset: https://www.kaggle.com/datasets/mczielinski/bitcoin-historical-data
Source: https://github.com/mczielinski/kaggle-bitcoin
"""

import os
import subprocess
import sys
import pandas as pd
from pathlib import Path

def check_kaggle_setup():
    """Check if Kaggle API is configured"""
    kaggle_dir = Path.home() / '.kaggle'
    kaggle_json = kaggle_dir / 'kaggle.json'
    
    if not kaggle_json.exists():
        print("Kaggle API not configured!")
        print("\nSetup instructions:")
        print("1. Go to https://www.kaggle.com/settings")
        print("2. Scroll to 'API' section")
        print("3. Click 'Create New API Token'")
        print(f"4. Move downloaded kaggle.json to {kaggle_dir}")
        
        if sys.platform == 'win32':
            print(f"   mkdir {kaggle_dir}")
            print(f"   move kaggle.json {kaggle_json}")
        else:
            print(f"   mkdir -p {kaggle_dir}")
            print(f"   mv ~/Downloads/kaggle.json {kaggle_json}")
            print(f"   chmod 600 {kaggle_json}")
        
        sys.exit(1)
    
    print(f"✓ Kaggle API configured at {kaggle_json}")

def install_kaggle():
    """Install kaggle package if not present"""
    try:
        import kaggle
        print("✓ Kaggle package installed")
    except ImportError:
        print("Installing kaggle package...")
        subprocess.check_call([sys.executable, "-m", "pip", "install", "kaggle"])
        print("✓ Kaggle package installed")

def download_dataset(output_dir="data/historical"):
    """Download the Bitcoin historical dataset from Kaggle"""
    import kaggle
    
    dataset = "mczielinski/bitcoin-historical-data"
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)
    
    print(f"\nDownloading dataset: {dataset}")
    print(f"Output directory: {output_path.absolute()}")
    
    # Download dataset
    kaggle.api.dataset_download_files(
        dataset,
        path=output_path,
        unzip=True
    )
    
    print(f"✓ Dataset downloaded to {output_path}")
    
    # List downloaded files
    files = list(output_path.glob("*.csv"))
    print(f"\nDownloaded files:")
    for file in files:
        size_mb = file.stat().st_size / (1024 * 1024)
        print(f"  - {file.name} ({size_mb:.2f} MB)")
    
    return files

def convert_to_candlecore_format(csv_file, output_file=None, recent_days=365, resample='5m'):
    """
    Convert Kaggle Bitcoin CSV to Candlecore format with size limits
    
    Args:
        csv_file: Input CSV file
        output_file: Output path (optional)
        recent_days: Keep only last N days (None for all data)
        resample: Resample interval ('1m', '5m', '15m', '1h', '4h', '1d')
    """
    print(f"\nConverting {csv_file.name} to Candlecore format...")
    print(f"  Settings: Last {recent_days} days, {resample} intervals")
    
    df = pd.read_csv(csv_file)
    print(f"  Original: {len(df):,} rows (~{csv_file.stat().st_size / (1024*1024):.1f} MB)")
    
    # Convert Unix timestamp to datetime
    df['timestamp'] = pd.to_datetime(df['Timestamp'], unit='s')
    
    # Filter to recent days if specified
    if recent_days:
        cutoff_date = df['timestamp'].max() - pd.Timedelta(days=recent_days)
        df = df[df['timestamp'] >= cutoff_date]
        print(f"  After date filter: {len(df):,} rows")
    
    # Select columns
    df = df[['timestamp', 'Open', 'High', 'Low', 'Close', 'Volume_(BTC)']].copy()
    df.columns = ['timestamp', 'open', 'high', 'low', 'close', 'volume']
    
    # Remove NaN
    df = df.dropna()
    
    # Resample if not 1-minute
    if resample != '1m':
        df.set_index('timestamp', inplace=True)
        df = df.resample(resample).agg({
            'open': 'first',
            'high': 'max',
            'low': 'min',
            'close': 'last',
            'volume': 'sum'
        }).dropna()
        df.reset_index(inplace=True)
        print(f"  After {resample} resample: {len(df):,} rows")
    
    # Convert timestamp to ISO format
    df['timestamp'] = df['timestamp'].dt.strftime('%Y-%m-%dT%H:%M:%SZ')
    
    if output_file is None:
        interval_suffix = resample.replace('m', 'min').replace('h', 'h').replace('d', 'd')
        output_file = csv_file.parent / f"bitcoin_{resample}.csv"
    
    df.to_csv(output_file, index=False)
    size_mb = output_file.stat().st_size / (1024 * 1024)
    print(f"  ✓ Saved to {output_file.name} ({size_mb:.1f} MB)")
    
    # Show stats
    print(f"\n  Data range:")
    print(f"    From: {df['timestamp'].iloc[0]}")
    print(f"    To:   {df['timestamp'].iloc[-1]}")
    print(f"    Total candles: {len(df):,}")
    
    return output_file

def main():
    print("=" * 60)
    print("Bitcoin Historical Data Downloader")
    print("=" * 60)
    
    # Configuration
    RECENT_DAYS = 365  # Last 1 year of data
    RESAMPLE = '5m'     # 5-minute intervals
    # Options: '1m', '5m', '15m', '1h', '4h', '1d'
    
    print(f"\nConfiguration:")
    print(f"  Time range: Last {RECENT_DAYS} days")
    print(f"  Interval: {RESAMPLE}")
    print(f"  Expected size: ~50 MB")
    
    # Check setup
    check_kaggle_setup()
    install_kaggle()
    
    # Download dataset
    files = download_dataset()
    
    # Convert to Candlecore format
    print("\n" + "=" * 60)
    print("Converting to Candlecore Format")
    print("=" * 60)
    
    for csv_file in files:
        if csv_file.name.endswith('.csv'):
            convert_to_candlecore_format(
                csv_file,
                recent_days=RECENT_DAYS,
                resample=RESAMPLE
            )
    
    print("\n" + "=" * 60)
    print("✓ Complete!")
    print("=" * 60)
    print("\nYou can now use the converted CSV files with Candlecore.")
    print("The data includes 1-minute Bitcoin prices from 2012 to present.")

if __name__ == "__main__":
    main()
