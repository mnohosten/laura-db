/**
 * Index Management Example
 *
 * Demonstrates index creation and management with the LauraDB Node.js client
 */

const { createClient } = require('../index');

async function main() {
  console.log('Index Management Example\n');

  const client = createClient({
    host: 'localhost',
    port: 8080
  });

  try {
    const users = client.collection('users');
    const indexes = users.indexes();

    // Insert sample data
    console.log('1. Inserting sample data...');
    await users.insertMany([
      {
        name: 'Alice',
        email: 'alice@example.com',
        age: 30,
        city: 'New York',
        location: { type: 'Point', coordinates: [-74.006, 40.7128] },
        skills: ['JavaScript', 'Node.js'],
        createdAt: new Date().toISOString()
      },
      {
        name: 'Bob',
        email: 'bob@example.com',
        age: 25,
        city: 'San Francisco',
        location: { type: 'Point', coordinates: [-122.4194, 37.7749] },
        skills: ['Python', 'Django'],
        createdAt: new Date().toISOString()
      },
      {
        name: 'Carol',
        email: 'carol@example.com',
        age: 35,
        city: 'Boston',
        location: { type: 'Point', coordinates: [-71.0589, 42.3601] },
        skills: ['Java', 'Spring'],
        createdAt: new Date().toISOString()
      }
    ]);
    console.log('   Inserted 3 users\n');

    // Create B+ tree index
    console.log('2. Creating B+ tree index on email (unique)...');
    await indexes.create('email', { unique: true });
    console.log('   ✓ Created email index\n');

    // Create compound index
    console.log('3. Creating compound index on city and age...');
    await indexes.createCompound({ city: 1, age: -1 });
    console.log('   ✓ Created compound index\n');

    // Create text index
    console.log('4. Creating text index on skills...');
    await indexes.createText(['skills']);
    console.log('   ✓ Created text index\n');

    // Create geospatial index
    console.log('5. Creating geospatial index on location...');
    await indexes.createGeo('location', '2dsphere');
    console.log('   ✓ Created geo index\n');

    // Create TTL index (for automatic expiration)
    console.log('6. Creating TTL index on createdAt (24 hour expiry)...');
    await indexes.createTTL('createdAt', 86400); // 24 hours
    console.log('   ✓ Created TTL index\n');

    // Create partial index (conditional indexing)
    console.log('7. Creating partial index on age (only for age > 25)...');
    await indexes.createPartial('age', { age: { $gt: 25 } });
    console.log('   ✓ Created partial index\n');

    // List all indexes
    console.log('8. Listing all indexes...');
    const allIndexes = await indexes.list();
    console.log(`   Total indexes: ${allIndexes.length}`);
    allIndexes.forEach((idx, i) => {
      console.log(`     ${i + 1}. ${idx.name} (${idx.type})`);
      if (idx.fields) {
        console.log(`        Fields: ${JSON.stringify(idx.fields)}`);
      }
      if (idx.unique) {
        console.log(`        Unique: true`);
      }
    });
    console.log('');

    // Query with index
    console.log('9. Querying with index usage...');
    const plan = await users.find()
      .filter({ age: { $gt: 25 } })
      .explain();
    console.log('   Query Plan:');
    console.log(`     Index Used: ${plan.index_used || 'none'}`);
    console.log(`     Execution Time: ${plan.execution_time_ms || 'N/A'}ms`);
    console.log('');

    // Perform queries using indexes
    console.log('10. Performing indexed queries...');

    // Query by email (unique index)
    const userByEmail = await users.find()
      .filter({ email: 'alice@example.com' })
      .first();
    console.log(`    ✓ Found user by email: ${userByEmail?.name}`);

    // Query by compound index (city + age)
    const usersByCity = await users.find()
      .filter({ city: 'New York', age: { $gte: 25 } })
      .execute();
    console.log(`    ✓ Found ${usersByCity.length} users in New York (age >= 25)`);

    console.log('');

    // Drop an index
    console.log('11. Dropping age index...');
    await indexes.drop('age');
    console.log('    ✓ Dropped age index\n');

    // Clean up - delete test data and remaining indexes
    console.log('12. Cleaning up...');
    await users.find().filter({}).delete();
    console.log('    ✓ Deleted all test users');

    // Note: In production, you might want to keep indexes
    // Here we're cleaning up for the demo
    const finalIndexes = await indexes.list();
    for (const idx of finalIndexes) {
      if (idx.name !== '_id_') { // Don't drop the default _id index
        try {
          await indexes.drop(idx.name);
        } catch (err) {
          // Some index types might need different drop methods
          console.log(`    Note: Could not drop ${idx.name} (${err.message})`);
        }
      }
    }
    console.log('    ✓ Cleaned up indexes\n');

    console.log('✓ Index management examples completed successfully!');

  } catch (err) {
    console.error('\n✗ Error:', err.message);
    if (err.apiError) {
      console.error('   API Error:', err.apiError);
    }
    if (err.stack) {
      console.error('   Stack:', err.stack);
    }
    process.exit(1);
  } finally {
    client.close();
    console.log('\nClient closed.');
  }
}

// Run the example
if (require.main === module) {
  main().catch(console.error);
}

module.exports = main;
