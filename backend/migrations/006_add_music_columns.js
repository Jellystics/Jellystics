exports.up = async function up(knex) {
  await knex.raw(`ALTER TABLE jf_library_items ADD COLUMN IF NOT EXISTS "AlbumId" text`);
  await knex.raw(`ALTER TABLE jf_library_items ADD COLUMN IF NOT EXISTS "Album" text`);
  await knex.raw(`ALTER TABLE jf_library_items ADD COLUMN IF NOT EXISTS "AlbumArtist" text`);
  await knex.raw(`ALTER TABLE jf_library_items ADD COLUMN IF NOT EXISTS "IndexNumber" integer`);
};

exports.down = async function down(knex) {
  await knex.schema.alterTable('jf_library_items', (table) => {
    table.dropColumn('AlbumId');
    table.dropColumn('Album');
    table.dropColumn('AlbumArtist');
    table.dropColumn('IndexNumber');
  });
};
