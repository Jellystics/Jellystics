exports.up = async function up(knex) {
  // Add imported flag to distinguish plugin-imported rows from real-time sessions
  await knex.schema.alterTable("jf_playback_activity", (table) => {
    table.boolean("imported").defaultTo(false);
  });

  // Remove any corrupted rows inserted with MD5 hashes instead of numeric SQLite rowids
  await knex.raw(`DELETE FROM jf_playback_reporting_plugin_data WHERE "rowid" !~ '^[0-9]+$'`);
};

exports.down = async function down(knex) {
  await knex.schema.alterTable("jf_playback_activity", (table) => {
    table.dropColumn("imported");
  });
};
