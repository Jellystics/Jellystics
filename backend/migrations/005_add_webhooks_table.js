exports.up = async function up(knex) {
  await knex.schema.createTableIfNotExists('webhooks', (table) => {
    table.increments('id').primary();
    table.text('name').notNullable();
    table.text('url').notNullable();
    table.text('method').defaultTo('POST');
    table.text('trigger_type').notNullable(); // 'event' | 'scheduled'
    table.text('event_type');
    table.text('schedule');
    table.boolean('enabled').defaultTo(true);
    table.jsonb('headers').defaultTo('{}');
    table.jsonb('payload').defaultTo('{}');
    table.boolean('retry_on_failure').defaultTo(false);
    table.integer('max_retries').defaultTo(3);
    table.timestamp('last_triggered');
    table.timestamps(true, true);
  });
};

exports.down = async function down(knex) {
  await knex.schema.dropTableIfExists('webhooks');
};
