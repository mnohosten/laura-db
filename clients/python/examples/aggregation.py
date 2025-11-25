#!/usr/bin/env python3
"""
LauraDB Python Client - Aggregation Pipeline Example

This example demonstrates:
- Building aggregation pipelines
- Using $match, $group, $project, $sort, $limit stages
- Aggregation operators ($sum, $avg, $min, $max, etc.)
- Complex grouping and transformations
"""

from lauradb import Client, Aggregation
import random
from datetime import datetime, timedelta


def main():
    print("=" * 60)
    print("LauraDB Python Client - Aggregation Example")
    print("=" * 60)

    # Connect to LauraDB
    client = Client(host='localhost', port=8080)

    if not client.ping():
        print("✗ Failed to connect to LauraDB server")
        return

    print("✓ Connected to LauraDB server")

    # Create collection
    collection_name = 'sales'
    print(f"\nCreating collection '{collection_name}'...")
    client.create_collection(collection_name)
    sales = client.collection(collection_name)

    # Generate sample sales data
    print("\nGenerating sample data...")
    cities = ['New York', 'San Francisco', 'Seattle', 'Boston', 'Austin']
    products = ['Laptop', 'Phone', 'Tablet', 'Monitor', 'Keyboard', 'Mouse']
    categories = ['Electronics', 'Accessories']

    sales_data = []
    base_date = datetime.now() - timedelta(days=30)

    for i in range(100):
        sale = {
            'orderId': f'ORD-{1000 + i}',
            'product': random.choice(products),
            'category': 'Electronics' if random.choice(products) in ['Laptop', 'Phone', 'Tablet', 'Monitor'] else 'Accessories',
            'quantity': random.randint(1, 5),
            'price': random.randint(50, 1500),
            'city': random.choice(cities),
            'date': (base_date + timedelta(days=random.randint(0, 30))).isoformat(),
            'customerId': f'CUST-{random.randint(1, 50)}'
        }
        sale['total'] = sale['quantity'] * sale['price']
        sales_data.append(sale)

    sales.insert_many(sales_data)
    print(f"Inserted {len(sales_data)} sales records")

    # Example 1: Group by city and calculate totals
    print("\n1. GROUP BY CITY - Total Sales")
    print("-" * 60)
    agg = Aggregation()
    pipeline = [
        agg.group(
            '$city',
            {
                'totalSales': agg.sum('$total'),
                'avgOrderValue': agg.avg('$total'),
                'orderCount': agg.count()
            }
        ),
        agg.sort({'totalSales': -1})
    ]

    results = sales.aggregate(pipeline)
    print("Sales by city:")
    for result in results:
        print(f"  {result['_id']}: ${result['totalSales']:,} "
              f"({result['orderCount']} orders, avg: ${result['avgOrderValue']:.2f})")

    # Example 2: Group by product with statistics
    print("\n2. GROUP BY PRODUCT - Statistics")
    print("-" * 60)
    pipeline = [
        agg.group(
            '$product',
            {
                'totalRevenue': agg.sum('$total'),
                'unitsSold': agg.sum('$quantity'),
                'avgPrice': agg.avg('$price'),
                'minPrice': agg.min_('$price'),
                'maxPrice': agg.max_('$price'),
                'orderCount': agg.count()
            }
        ),
        agg.sort({'totalRevenue': -1}),
        agg.limit(5)
    ]

    results = sales.aggregate(pipeline)
    print("Top 5 products by revenue:")
    for result in results:
        print(f"  {result['_id']}:")
        print(f"    Revenue: ${result['totalRevenue']:,}")
        print(f"    Units sold: {result['unitsSold']}")
        print(f"    Avg price: ${result['avgPrice']:.2f}")
        print(f"    Price range: ${result['minPrice']} - ${result['maxPrice']}")

    # Example 3: Filter and group - High value orders
    print("\n3. FILTER AND GROUP - High Value Orders")
    print("-" * 60)
    pipeline = [
        agg.match({'total': {'$gte': 500}}),
        agg.group(
            '$city',
            {
                'highValueOrders': agg.count(),
                'totalValue': agg.sum('$total'),
                'products': agg.push('$product')
            }
        ),
        agg.sort({'totalValue': -1})
    ]

    results = sales.aggregate(pipeline)
    print("High value orders (>=$500) by city:")
    for result in results:
        print(f"  {result['_id']}: {result['highValueOrders']} orders, "
              f"${result['totalValue']:,}")

    # Example 4: Category analysis
    print("\n4. CATEGORY ANALYSIS")
    print("-" * 60)
    pipeline = [
        agg.group(
            '$category',
            {
                'totalRevenue': agg.sum('$total'),
                'avgOrderValue': agg.avg('$total'),
                'orderCount': agg.count(),
                'uniqueProducts': agg.add_to_set('$product')
            }
        ),
        agg.sort({'totalRevenue': -1})
    ]

    results = sales.aggregate(pipeline)
    print("Sales by category:")
    for result in results:
        print(f"  {result['_id']}:")
        print(f"    Revenue: ${result['totalRevenue']:,}")
        print(f"    Orders: {result['orderCount']}")
        print(f"    Avg order: ${result['avgOrderValue']:.2f}")
        print(f"    Product variety: {len(result['uniqueProducts'])} products")

    # Example 5: Customer analysis
    print("\n5. CUSTOMER ANALYSIS - Top Customers")
    print("-" * 60)
    pipeline = [
        agg.group(
            '$customerId',
            {
                'totalSpent': agg.sum('$total'),
                'orderCount': agg.count(),
                'avgOrderValue': agg.avg('$total'),
                'cities': agg.add_to_set('$city')
            }
        ),
        agg.sort({'totalSpent': -1}),
        agg.limit(10)
    ]

    results = sales.aggregate(pipeline)
    print("Top 10 customers by spending:")
    for i, result in enumerate(results, 1):
        print(f"  {i}. {result['_id']}: ${result['totalSpent']:,} "
              f"({result['orderCount']} orders, {len(result['cities'])} cities)")

    # Example 6: Product and city combination
    print("\n6. PRODUCT-CITY COMBINATION")
    print("-" * 60)
    pipeline = [
        agg.group(
            {'product': '$product', 'city': '$city'},
            {
                'totalSales': agg.sum('$total'),
                'quantity': agg.sum('$quantity')
            }
        ),
        agg.sort({'totalSales': -1}),
        agg.limit(10)
    ]

    results = sales.aggregate(pipeline)
    print("Top 10 product-city combinations:")
    for result in results:
        print(f"  {result['_id']['product']} in {result['_id']['city']}: "
              f"${result['totalSales']:,} ({result['quantity']} units)")

    # Example 7: Project stage - Transform data
    print("\n7. PROJECT - Transform Data")
    print("-" * 60)
    pipeline = [
        agg.match({'city': 'New York'}),
        agg.project({
            'orderId': 1,
            'product': 1,
            'total': 1,
            'isHighValue': agg.cond(
                {'$gte': ['$total', 500]},
                'High',
                'Normal'
            )
        }),
        agg.sort({'total': -1}),
        agg.limit(5)
    ]

    results = sales.aggregate(pipeline)
    print("New York orders with value classification:")
    for result in results:
        print(f"  {result['orderId']}: {result['product']} - "
              f"${result['total']} ({result['isHighValue']} value)")

    # Example 8: Multi-stage complex pipeline
    print("\n8. COMPLEX PIPELINE - Sales Performance")
    print("-" * 60)
    pipeline = [
        # Filter recent high-value orders
        agg.match({'total': {'$gte': 300}}),
        # Group by city
        agg.group(
            '$city',
            {
                'revenue': agg.sum('$total'),
                'orders': agg.count(),
                'avgOrder': agg.avg('$total')
            }
        ),
        # Add computed fields
        agg.project({
            'city': '$_id',
            'revenue': 1,
            'orders': 1,
            'avgOrder': 1,
            '_id': 0
        }),
        # Sort by revenue
        agg.sort({'revenue': -1}),
        # Top 3 cities
        agg.limit(3)
    ]

    results = sales.aggregate(pipeline)
    print("Top 3 cities by high-value order revenue (>=$300):")
    for i, result in enumerate(results, 1):
        print(f"  {i}. {result['city']}:")
        print(f"     Revenue: ${result['revenue']:,}")
        print(f"     Orders: {result['orders']}")
        print(f"     Avg: ${result['avgOrder']:.2f}")

    # Example 9: Summary statistics
    print("\n9. OVERALL SUMMARY STATISTICS")
    print("-" * 60)
    pipeline = [
        agg.group(
            None,  # Group all documents
            {
                'totalRevenue': agg.sum('$total'),
                'totalOrders': agg.count(),
                'avgOrderValue': agg.avg('$total'),
                'minOrder': agg.min_('$total'),
                'maxOrder': agg.max_('$total'),
                'uniqueCustomers': agg.add_to_set('$customerId'),
                'uniqueCities': agg.add_to_set('$city'),
                'uniqueProducts': agg.add_to_set('$product')
            }
        )
    ]

    results = sales.aggregate(pipeline)
    if results:
        stats = results[0]
        print("Overall sales statistics:")
        print(f"  Total revenue: ${stats['totalRevenue']:,}")
        print(f"  Total orders: {stats['totalOrders']}")
        print(f"  Avg order value: ${stats['avgOrderValue']:.2f}")
        print(f"  Order range: ${stats['minOrder']} - ${stats['maxOrder']}")
        print(f"  Unique customers: {len(stats['uniqueCustomers'])}")
        print(f"  Cities served: {len(stats['uniqueCities'])}")
        print(f"  Products sold: {len(stats['uniqueProducts'])}")

    # Cleanup
    print("\n10. CLEANUP")
    print("-" * 60)
    client.drop_collection(collection_name)
    print(f"Dropped collection '{collection_name}'")

    client.close()
    print("\n✓ Example completed successfully")
    print("=" * 60)


if __name__ == '__main__':
    main()
