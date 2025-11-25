"""
Tests for LauraDB Query builder
"""
import pytest
from lauradb import Query, UpdateBuilder


class TestQuery:
    """Test Query builder."""

    def test_eq(self):
        """Test $eq operator."""
        result = Query.eq('age', 30)
        assert result == {'age': {'$eq': 30}}

    def test_ne(self):
        """Test $ne operator."""
        result = Query.ne('status', 'inactive')
        assert result == {'status': {'$ne': 'inactive'}}

    def test_gt(self):
        """Test $gt operator."""
        result = Query.gt('age', 25)
        assert result == {'age': {'$gt': 25}}

    def test_gte(self):
        """Test $gte operator."""
        result = Query.gte('age', 18)
        assert result == {'age': {'$gte': 18}}

    def test_lt(self):
        """Test $lt operator."""
        result = Query.lt('age', 65)
        assert result == {'age': {'$lt': 65}}

    def test_lte(self):
        """Test $lte operator."""
        result = Query.lte('price', 100)
        assert result == {'price': {'$lte': 100}}

    def test_in(self):
        """Test $in operator."""
        result = Query.in_('status', ['active', 'pending'])
        assert result == {'status': {'$in': ['active', 'pending']}}

    def test_nin(self):
        """Test $nin operator."""
        result = Query.nin('role', ['guest', 'anonymous'])
        assert result == {'role': {'$nin': ['guest', 'anonymous']}}

    def test_and(self):
        """Test $and operator."""
        result = Query.and_(
            Query.gte('age', 18),
            Query.lt('age', 65)
        )
        assert result == {
            '$and': [
                {'age': {'$gte': 18}},
                {'age': {'$lt': 65}}
            ]
        }

    def test_or(self):
        """Test $or operator."""
        result = Query.or_(
            Query.eq('role', 'admin'),
            Query.eq('role', 'moderator')
        )
        assert result == {
            '$or': [
                {'role': {'$eq': 'admin'}},
                {'role': {'$eq': 'moderator'}}
            ]
        }

    def test_not(self):
        """Test $not operator."""
        result = Query.not_(Query.eq('active', True))
        assert result == {'$not': {'active': {'$eq': True}}}

    def test_exists(self):
        """Test $exists operator."""
        result = Query.exists('email', True)
        assert result == {'email': {'$exists': True}}

    def test_type(self):
        """Test $type operator."""
        result = Query.type_('age', 'number')
        assert result == {'age': {'$type': 'number'}}

    def test_all(self):
        """Test $all operator."""
        result = Query.all_('tags', ['python', 'database'])
        assert result == {'tags': {'$all': ['python', 'database']}}

    def test_elem_match(self):
        """Test $elemMatch operator."""
        result = Query.elem_match('scores', {'$gte': 80})
        assert result == {'scores': {'$elemMatch': {'$gte': 80}}}

    def test_size(self):
        """Test $size operator."""
        result = Query.size('tags', 3)
        assert result == {'tags': {'$size': 3}}

    def test_regex(self):
        """Test $regex operator."""
        result = Query.regex('name', '^A.*')
        assert result == {'name': {'$regex': '^A.*'}}

    def test_text(self):
        """Test $text operator."""
        result = Query.text('python database')
        assert result == {'$text': {'$search': 'python database'}}

    def test_near(self):
        """Test $near operator."""
        result = Query.near('location', -73.9857, 40.7580, 5000)
        assert result == {
            'location': {
                '$near': {
                    'coordinates': [-73.9857, 40.7580],
                    'maxDistance': 5000
                }
            }
        }

    def test_geo_within(self):
        """Test $geoWithin operator."""
        polygon = [[0, 0], [0, 10], [10, 10], [10, 0], [0, 0]]
        result = Query.geo_within('location', polygon)
        assert result == {
            'location': {
                '$geoWithin': {
                    'coordinates': polygon
                }
            }
        }

    def test_complex_query(self):
        """Test building a complex query."""
        query = Query.and_(
            Query.gte('age', 18),
            Query.lt('age', 65),
            Query.in_('role', ['user', 'admin']),
            Query.exists('email', True)
        )

        assert '$and' in query
        assert len(query['$and']) == 4


class TestUpdateBuilder:
    """Test UpdateBuilder."""

    def test_set(self):
        """Test $set operator."""
        result = UpdateBuilder.set('name', 'Alice')
        assert result == {'$set': {'name': 'Alice'}}

    def test_unset(self):
        """Test $unset operator."""
        result = UpdateBuilder.unset('tempField')
        assert result == {'$unset': {'tempField': ''}}

    def test_rename(self):
        """Test $rename operator."""
        result = UpdateBuilder.rename('oldName', 'newName')
        assert result == {'$rename': {'oldName': 'newName'}}

    def test_current_date(self):
        """Test $currentDate operator."""
        result = UpdateBuilder.current_date('updatedAt')
        assert result == {'$currentDate': {'updatedAt': True}}

    def test_inc(self):
        """Test $inc operator."""
        result = UpdateBuilder.inc('views', 1)
        assert result == {'$inc': {'views': 1}}

    def test_mul(self):
        """Test $mul operator."""
        result = UpdateBuilder.mul('price', 1.1)
        assert result == {'$mul': {'price': 1.1}}

    def test_min(self):
        """Test $min operator."""
        result = UpdateBuilder.min_('score', 100)
        assert result == {'$min': {'score': 100}}

    def test_max(self):
        """Test $max operator."""
        result = UpdateBuilder.max_('score', 0)
        assert result == {'$max': {'score': 0}}

    def test_push(self):
        """Test $push operator."""
        result = UpdateBuilder.push('tags', 'python')
        assert result == {'$push': {'tags': 'python'}}

    def test_pull(self):
        """Test $pull operator."""
        result = UpdateBuilder.pull('tags', 'deprecated')
        assert result == {'$pull': {'tags': 'deprecated'}}

    def test_pull_all(self):
        """Test $pullAll operator."""
        result = UpdateBuilder.pull_all('tags', ['old', 'deprecated'])
        assert result == {'$pullAll': {'tags': ['old', 'deprecated']}}

    def test_add_to_set(self):
        """Test $addToSet operator."""
        result = UpdateBuilder.add_to_set('tags', 'python')
        assert result == {'$addToSet': {'tags': 'python'}}

    def test_pop(self):
        """Test $pop operator."""
        result = UpdateBuilder.pop('items', -1)
        assert result == {'$pop': {'items': -1}}

    def test_bit_and(self):
        """Test $bit and operator."""
        result = UpdateBuilder.bit_and('flags', 0b1010)
        assert result == {'$bit': {'flags': {'and': 0b1010}}}

    def test_bit_or(self):
        """Test $bit or operator."""
        result = UpdateBuilder.bit_or('flags', 0b0101)
        assert result == {'$bit': {'flags': {'or': 0b0101}}}

    def test_bit_xor(self):
        """Test $bit xor operator."""
        result = UpdateBuilder.bit_xor('flags', 0b1111)
        assert result == {'$bit': {'flags': {'xor': 0b1111}}}

    def test_combine(self):
        """Test combining multiple update operations."""
        result = UpdateBuilder.combine(
            UpdateBuilder.set('name', 'Alice'),
            UpdateBuilder.inc('views', 1),
            UpdateBuilder.push('tags', 'python')
        )

        assert '$set' in result
        assert '$inc' in result
        assert '$push' in result
        assert result['$set'] == {'name': 'Alice'}
        assert result['$inc'] == {'views': 1}
        assert result['$push'] == {'tags': 'python'}

    def test_combine_same_operator(self):
        """Test combining operations with the same operator."""
        result = UpdateBuilder.combine(
            UpdateBuilder.set('name', 'Alice'),
            UpdateBuilder.set('age', 30)
        )

        assert result == {'$set': {'name': 'Alice', 'age': 30}}


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
