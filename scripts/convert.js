const fs = require('fs');

['1m', '5m', '15m'].forEach((interval) => {
  const data = JSON.parse(
    fs.readFileSync(`data/historical/bitcoin_${interval}.json`, 'utf8')
  );
  const csv = ['timestamp,open,high,low,close,volume'];

  data.forEach((k) => {
    const timestamp = new Date(k[0]).toISOString();
    csv.push(`${timestamp},${k[1]},${k[2]},${k[3]},${k[4]},${k[5]}`);
  });

  fs.writeFileSync(`data/historical/bitcoin_${interval}.csv`, csv.join('\n'));
  console.log(`Converted ${interval}: ${data.length} candles`);
});
