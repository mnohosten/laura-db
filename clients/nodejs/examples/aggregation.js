/**
 * Aggregation Pipeline Example
 *
 * Demonstrates aggregation operations with the LauraDB Node.js client
 */

const { createClient } = require('../index');

async function main() {
  console.log('Aggregation Pipeline Example\n');

  const client = createClient({
    host: 'localhost',
    port: 8080
  });

  try {
    const users = client.collection('users');

    // Insert sample data
    console.log('1. Inserting sample data...');
    await users.insertMany([
      { name: 'Alice', age: 30, city: 'New York', salary: 75000 },
      { name: 'Bob', age: 25, city: 'New York', salary: 60000 },
      { name: 'Carol', age: 35, city: 'San Francisco', salary: 95000 },
      { name: 'Dave', age: 28, city: 'San Francisco', salary: 80000 },
      { name: 'Eve', age: 32, city: 'Boston', salary: 70000 },
      { name: 'Frank', age: 45, city: 'Boston', salary: 110000 }
    ]);
    console.log('   Inserted 6 users\n');

    // Example 1: Group by city with statistics
    console.log('2. Grouping by city with statistics...');
    const cityStats = await users.aggregate()
      .match({ age: { $gt: 20 } })
      .group({
        _id: '$city',
        totalUsers: { $sum: 1 },
        avgAge: { $avg: '$age' },
        avgSalary: { $avg: '$salary' },
        minSalary: { $min: '$salary' },
        maxSalary: { $max: '$salary' }
      })
      .sort({ avgSalary: -1 })
      .execute();

    console.log('   City Statistics:');
    cityStats.forEach(stat => {
      console.log(`     ${stat._id}:`);
      console.log(`       Users: ${stat.totalUsers}`);
      console.log(`       Avg Age: ${stat.avgAge.toFixed(1)}`);
      console.log(`       Avg Salary: $${stat.avgSalary.toFixed(0)}`);
      console.log(`       Salary Range: $${stat.minSalary} - $${stat.maxSalary}`);
      console.log('');
    });

    // Example 2: Project and transform
    console.log('3. Projecting and transforming data...');
    const transformed = await users.aggregate()
      .match({ age: { $gte: 30 } })
      .project({
        name: 1,
        age: 1,
        city: 1,
        salary: 1
      })
      .sort({ salary: -1 })
      .limit(3)
      .execute();

    console.log('   Top 3 earners (age >= 30):');
    transformed.forEach((user, idx) => {
      console.log(`     ${idx + 1}. ${user.name} (${user.age}) - $${user.salary}`);
    });
    console.log('');

    // Example 3: Complex pipeline with multiple stages
    console.log('4. Complex pipeline - Users by age group...');
    const ageGroups = await users.aggregate()
      .group({
        _id: null,
        youngsters: {
          $sum: 1  // This would ideally use $cond in MongoDB
        },
        middleAge: {
          $sum: 1
        },
        seniors: {
          $sum: 1
        },
        totalUsers: { $sum: 1 },
        avgAge: { $avg: '$age' }
      })
      .execute();

    console.log('   Age Group Analysis:');
    console.log(`     Total Users: ${ageGroups[0].totalUsers}`);
    console.log(`     Average Age: ${ageGroups[0].avgAge.toFixed(1)}`);
    console.log('');

    // Example 4: Salary analysis
    console.log('5. Salary analysis by city...');
    const salaryAnalysis = await users.aggregate()
      .group({
        _id: '$city',
        users: { $push: '$name' },
        totalSalary: { $sum: '$salary' },
        avgSalary: { $avg: '$salary' },
        count: { $sum: 1 }
      })
      .sort({ totalSalary: -1 })
      .execute();

    console.log('   Salary by City:');
    salaryAnalysis.forEach(city => {
      console.log(`     ${city._id}:`);
      console.log(`       Users: ${city.users.join(', ')}`);
      console.log(`       Total Payroll: $${city.totalSalary.toLocaleString()}`);
      console.log(`       Avg Salary: $${city.avgSalary.toFixed(0)}`);
      console.log('');
    });

    // Clean up - delete test data
    console.log('6. Cleaning up test data...');
    const deleted = await users.find()
      .filter({})
      .delete();
    console.log(`   Deleted ${deleted} users\n`);

    console.log('✓ Aggregation examples completed successfully!');

  } catch (err) {
    console.error('\n✗ Error:', err.message);
    if (err.apiError) {
      console.error('   API Error:', err.apiError);
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
