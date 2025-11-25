"""
Tests for LauraDB Aggregation builder
"""
import pytest
from lauradb import Aggregation


class TestAggregation:
    """Test Aggregation builder."""

    def test_match(self):
        """Test $match stage."""
        stage = Aggregation.match({'age': {'$gte': 18}})
        assert stage == {'$match': {'age': {'$gte': 18}}}

    def test_group(self):
        """Test $group stage."""
        stage = Aggregation.group(
            '$city',
            {
                'avgAge': Aggregation.avg('$age'),
                'count': Aggregation.count()
            }
        )

        assert '$group' in stage
        assert stage['$group']['_id'] == '$city'
        assert 'avgAge' in stage['$group']
        assert 'count' in stage['$group']

    def test_group_compound_key(self):
        """Test $group with compound key."""
        stage = Aggregation.group(
            {'city': '$city', 'year': '$year'},
            {'total': Aggregation.sum('$amount')}
        )

        assert stage['$group']['_id'] == {'city': '$city', 'year': '$year'}

    def test_project(self):
        """Test $project stage."""
        stage = Aggregation.project({
            'name': 1,
            'age': 1,
            'email': 1,
            '_id': 0
        })

        assert stage == {
            '$project': {
                'name': 1,
                'age': 1,
                'email': 1,
                '_id': 0
            }
        }

    def test_project_with_expressions(self):
        """Test $project with computed fields."""
        stage = Aggregation.project({
            'name': 1,
            'fullName': Aggregation.concat('$firstName', ' ', '$lastName')
        })

        assert '$concat' in stage['$project']['fullName']

    def test_sort(self):
        """Test $sort stage."""
        stage = Aggregation.sort({'age': -1, 'name': 1})
        assert stage == {'$sort': {'age': -1, 'name': 1}}

    def test_limit(self):
        """Test $limit stage."""
        stage = Aggregation.limit(10)
        assert stage == {'$limit': 10}

    def test_skip(self):
        """Test $skip stage."""
        stage = Aggregation.skip(20)
        assert stage == {'$skip': 20}

    def test_unwind(self):
        """Test $unwind stage."""
        stage = Aggregation.unwind('$tags')
        assert stage == {'$unwind': '$tags'}

    def test_unwind_preserve_null(self):
        """Test $unwind with preserveNullAndEmptyArrays."""
        stage = Aggregation.unwind('$tags', preserve_null=True)
        assert stage == {
            '$unwind': {
                'path': '$tags',
                'preserveNullAndEmptyArrays': True
            }
        }

    def test_lookup(self):
        """Test $lookup stage."""
        stage = Aggregation.lookup('orders', 'userId', '_id', 'userOrders')
        assert stage == {
            '$lookup': {
                'from': 'orders',
                'localField': 'userId',
                'foreignField': '_id',
                'as': 'userOrders'
            }
        }

    def test_sum(self):
        """Test $sum operator."""
        result = Aggregation.sum('$amount')
        assert result == {'$sum': '$amount'}

    def test_sum_constant(self):
        """Test $sum with constant for counting."""
        result = Aggregation.sum(1)
        assert result == {'$sum': 1}

    def test_avg(self):
        """Test $avg operator."""
        result = Aggregation.avg('$age')
        assert result == {'$avg': '$age'}

    def test_min(self):
        """Test $min operator."""
        result = Aggregation.min_('$price')
        assert result == {'$min': '$price'}

    def test_max(self):
        """Test $max operator."""
        result = Aggregation.max_('$price')
        assert result == {'$max': '$price'}

    def test_count(self):
        """Test $count operator."""
        result = Aggregation.count()
        assert result == {'$count': {}}

    def test_push(self):
        """Test $push operator."""
        result = Aggregation.push('$name')
        assert result == {'$push': '$name'}

    def test_add_to_set(self):
        """Test $addToSet operator."""
        result = Aggregation.add_to_set('$category')
        assert result == {'$addToSet': '$category'}

    def test_first(self):
        """Test $first operator."""
        result = Aggregation.first('$createdAt')
        assert result == {'$first': '$createdAt'}

    def test_last(self):
        """Test $last operator."""
        result = Aggregation.last('$updatedAt')
        assert result == {'$last': '$updatedAt'}

    def test_concat(self):
        """Test $concat operator."""
        result = Aggregation.concat('$firstName', ' ', '$lastName')
        assert result == {'$concat': ['$firstName', ' ', '$lastName']}

    def test_substring(self):
        """Test $substr operator."""
        result = Aggregation.substring('$name', 0, 5)
        assert result == {'$substr': ['$name', 0, 5]}

    def test_to_upper(self):
        """Test $toUpper operator."""
        result = Aggregation.to_upper('$name')
        assert result == {'$toUpper': '$name'}

    def test_to_lower(self):
        """Test $toLower operator."""
        result = Aggregation.to_lower('$email')
        assert result == {'$toLower': '$email'}

    def test_cond(self):
        """Test $cond operator."""
        result = Aggregation.cond(
            {'$gte': ['$age', 18]},
            'adult',
            'minor'
        )
        assert result == {
            '$cond': [
                {'$gte': ['$age', 18]},
                'adult',
                'minor'
            ]
        }

    def test_complete_pipeline(self):
        """Test building a complete aggregation pipeline."""
        pipeline = [
            Aggregation.match({'age': {'$gte': 18}}),
            Aggregation.group(
                '$city',
                {
                    'avgAge': Aggregation.avg('$age'),
                    'total': Aggregation.sum(1),
                    'names': Aggregation.push('$name')
                }
            ),
            Aggregation.sort({'avgAge': -1}),
            Aggregation.limit(10)
        ]

        assert len(pipeline) == 4
        assert '$match' in pipeline[0]
        assert '$group' in pipeline[1]
        assert '$sort' in pipeline[2]
        assert '$limit' in pipeline[3]


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
