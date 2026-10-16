// The starter pack is authored independently of frontend preview copy. This
// command intentionally validates it; it must never overwrite editorial work.
const fs = require('node:fs');
const rows = JSON.parse(fs.readFileSync('templates/quickstart/internal/feature/content/seed.json', 'utf8'));
const paths = new Set();
for (const row of rows) {
  if (paths.has(row.path)) throw new Error(`Duplicate path: ${row.path}`);
  paths.add(row.path);
  for (const locale of ['en', 'zh']) {
    const text = row.translations[locale];
    if (!text || !text.title?.trim() || !text.summary?.trim() || !text.body?.trim()) throw new Error(`Incomplete ${row.path} ${locale}`);
  }
}
console.log(`Validated ${rows.length} bilingual content records. Import explicitly from admin; no database was changed.`);
