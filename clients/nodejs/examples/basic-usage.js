/**
 * Basic Usage Example
 *
 * Demonstrates core CRUD operations with the LauraDB Node.js client
 */

const { createClient } = require('../index');

async function main() {
  // Create client
  console.log('Creating LauraDB client...');
  const client = createClient({
    host: 'localhost',
    port: 8080
  });

  try {
    // Check server health
    console.log('\n1. Checking server health...');
    const health = await client.health();
    console.log('   Status:', health.status);
    console.log('   Uptime:', health.uptime);

    // Get collection handle
    const users = client.collection('users');

    // Insert a document
    console.log('\n2. Inserting document...');
    const id = await users.insertOne({
      name: 'Alice Johnson',
      email: 'alice@example.com',
      age: 30,
      city: 'New York',
      skills: ['JavaScript', 'Node.js', 'MongoDB']
    });
    console.log('   Inserted ID:', id);

    // Find the document
    console.log('\n3. Finding document by ID...');
    const user = await users.findOne(id);
    console.log('   Found user:', JSON.stringify(user, null, 2));

    // Update the document
    console.log('\n4. Updating document...');
    await users.updateOne(id, {
      $set: { age: 31, city: 'San Francisco' },
      $push: { skills: 'React' }
    });
    console.log('   Updated successfully');

    // Find with query
    console.log('\n5. Querying documents...');
    const results = await users.find()
      .filter({ city: 'San Francisco' })
      .project({ name: 1, age: 1, city: 1 })
      .sort({ age: -1 })
      .execute();
    console.log('   Query results:', JSON.stringify(results, null, 2));

    // Count documents
    console.log('\n6. Counting documents...');
    const count = await users.count({ age: { $gt: 25 } });
    console.log('   Count (age > 25):', count);

    // Delete the document
    console.log('\n7. Deleting document...');
    await users.deleteOne(id);
    console.log('   Deleted successfully');

    // List collections
    console.log('\n8. Listing collections...');
    const collections = await client.listCollections();
    console.log('   Collections:', collections);

    // Get database stats
    console.log('\n9. Getting database statistics...');
    const stats = await client.stats();
    console.log('   Total collections:', stats.collections);
    console.log('   Active transactions:', stats.active_transactions);

    console.log('\n✓ All operations completed successfully!');

  } catch (err) {
    console.error('\n✗ Error:', err.message);
    if (err.apiError) {
      console.error('   API Error:', err.apiError);
    }
    process.exit(1);
  } finally {
    // Close client
    client.close();
    console.log('\nClient closed.');
  }
}

// Run the example
if (require.main === module) {
  main().catch(console.error);
}

module.exports = main;
