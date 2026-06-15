exports.up = async function up(knex) {
  await knex.raw(`
    CREATE OR REPLACE PROCEDURE jd_remove_orphaned_data()
    LANGUAGE plpgsql AS $$
    BEGIN
      DELETE FROM public.jf_item_info
      WHERE "Id" NOT IN (
        SELECT "Id" FROM public.jf_library_items
      );
    END;
    $$
  `);
};

exports.down = async function down(knex) {
  await knex.raw('DROP PROCEDURE IF EXISTS jd_remove_orphaned_data()');
};
