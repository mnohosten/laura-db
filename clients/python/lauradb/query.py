"""
LauraDB Query - Query builder and operators
"""
from typing import Any, Dict, List, Union


class Query:
    """
    Query builder for constructing MongoDB-style queries.

    This class provides methods for building complex queries using
    comparison, logical, array, and element operators.

    Example:
        >>> q = Query()
        >>> filter = q.and_(
        ...     q.gte('age', 25),
        ...     q.lt('age', 40),
        ...     q.eq('active', True)
        ... )
    """

    # Comparison operators
    @staticmethod
    def eq(field: str, value: Any) -> Dict[str, Any]:
        """Equal to ($eq)"""
        return {field: {"$eq": value}}

    @staticmethod
    def ne(field: str, value: Any) -> Dict[str, Any]:
        """Not equal to ($ne)"""
        return {field: {"$ne": value}}

    @staticmethod
    def gt(field: str, value: Any) -> Dict[str, Any]:
        """Greater than ($gt)"""
        return {field: {"$gt": value}}

    @staticmethod
    def gte(field: str, value: Any) -> Dict[str, Any]:
        """Greater than or equal to ($gte)"""
        return {field: {"$gte": value}}

    @staticmethod
    def lt(field: str, value: Any) -> Dict[str, Any]:
        """Less than ($lt)"""
        return {field: {"$lt": value}}

    @staticmethod
    def lte(field: str, value: Any) -> Dict[str, Any]:
        """Less than or equal to ($lte)"""
        return {field: {"$lte": value}}

    @staticmethod
    def in_(field: str, values: List[Any]) -> Dict[str, Any]:
        """Value in array ($in)"""
        return {field: {"$in": values}}

    @staticmethod
    def nin(field: str, values: List[Any]) -> Dict[str, Any]:
        """Value not in array ($nin)"""
        return {field: {"$nin": values}}

    # Logical operators
    @staticmethod
    def and_(*conditions: Dict[str, Any]) -> Dict[str, Any]:
        """Logical AND ($and)"""
        return {"$and": list(conditions)}

    @staticmethod
    def or_(*conditions: Dict[str, Any]) -> Dict[str, Any]:
        """Logical OR ($or)"""
        return {"$or": list(conditions)}

    @staticmethod
    def not_(condition: Dict[str, Any]) -> Dict[str, Any]:
        """Logical NOT ($not)"""
        return {"$not": condition}

    # Element operators
    @staticmethod
    def exists(field: str, exists: bool = True) -> Dict[str, Any]:
        """Field exists ($exists)"""
        return {field: {"$exists": exists}}

    @staticmethod
    def type_(field: str, type_name: str) -> Dict[str, Any]:
        """Field type ($type)"""
        return {field: {"$type": type_name}}

    # Array operators
    @staticmethod
    def all_(field: str, values: List[Any]) -> Dict[str, Any]:
        """Array contains all values ($all)"""
        return {field: {"$all": values}}

    @staticmethod
    def elem_match(field: str, condition: Dict[str, Any]) -> Dict[str, Any]:
        """Array element matches condition ($elemMatch)"""
        return {field: {"$elemMatch": condition}}

    @staticmethod
    def size(field: str, size: int) -> Dict[str, Any]:
        """Array has specific size ($size)"""
        return {field: {"$size": size}}

    # Evaluation operators
    @staticmethod
    def regex(field: str, pattern: str) -> Dict[str, Any]:
        """Regular expression match ($regex)"""
        return {field: {"$regex": pattern}}

    @staticmethod
    def text(search: str) -> Dict[str, Any]:
        """Text search ($text)"""
        return {"$text": {"$search": search}}

    # Geospatial operators
    @staticmethod
    def near(field: str, longitude: float, latitude: float, max_distance: float) -> Dict[str, Any]:
        """Near point ($near)"""
        return {
            field: {
                "$near": {
                    "coordinates": [longitude, latitude],
                    "maxDistance": max_distance
                }
            }
        }

    @staticmethod
    def geo_within(field: str, coordinates: List[List[float]]) -> Dict[str, Any]:
        """Within polygon ($geoWithin)"""
        return {
            field: {
                "$geoWithin": {
                    "coordinates": coordinates
                }
            }
        }


class UpdateBuilder:
    """
    Update operation builder for constructing update documents.

    This class provides methods for building update operations using
    field, numeric, and array update operators.

    Example:
        >>> u = UpdateBuilder()
        >>> update = u.combine(
        ...     u.set('name', 'Alice'),
        ...     u.inc('views', 1),
        ...     u.push('tags', 'python')
        ... )
    """

    # Field update operators
    @staticmethod
    def set(field: str, value: Any) -> Dict[str, Any]:
        """Set field value ($set)"""
        return {"$set": {field: value}}

    @staticmethod
    def unset(field: str) -> Dict[str, Any]:
        """Remove field ($unset)"""
        return {"$unset": {field: ""}}

    @staticmethod
    def rename(old_name: str, new_name: str) -> Dict[str, Any]:
        """Rename field ($rename)"""
        return {"$rename": {old_name: new_name}}

    @staticmethod
    def current_date(field: str) -> Dict[str, Any]:
        """Set to current date/time ($currentDate)"""
        return {"$currentDate": {field: True}}

    # Numeric update operators
    @staticmethod
    def inc(field: str, amount: Union[int, float]) -> Dict[str, Any]:
        """Increment numeric value ($inc)"""
        return {"$inc": {field: amount}}

    @staticmethod
    def mul(field: str, multiplier: Union[int, float]) -> Dict[str, Any]:
        """Multiply numeric value ($mul)"""
        return {"$mul": {field: multiplier}}

    @staticmethod
    def min_(field: str, value: Any) -> Dict[str, Any]:
        """Update if less than current ($min)"""
        return {"$min": {field: value}}

    @staticmethod
    def max_(field: str, value: Any) -> Dict[str, Any]:
        """Update if greater than current ($max)"""
        return {"$max": {field: value}}

    # Array update operators
    @staticmethod
    def push(field: str, value: Any) -> Dict[str, Any]:
        """Add to array ($push)"""
        return {"$push": {field: value}}

    @staticmethod
    def pull(field: str, value: Any) -> Dict[str, Any]:
        """Remove from array ($pull)"""
        return {"$pull": {field: value}}

    @staticmethod
    def pull_all(field: str, values: List[Any]) -> Dict[str, Any]:
        """Remove multiple values from array ($pullAll)"""
        return {"$pullAll": {field: values}}

    @staticmethod
    def add_to_set(field: str, value: Any) -> Dict[str, Any]:
        """Add unique value to array ($addToSet)"""
        return {"$addToSet": {field: value}}

    @staticmethod
    def pop(field: str, position: int = -1) -> Dict[str, Any]:
        """Remove first (-1) or last (1) array element ($pop)"""
        return {"$pop": {field: position}}

    # Bitwise operators
    @staticmethod
    def bit_and(field: str, value: int) -> Dict[str, Any]:
        """Bitwise AND ($bit and)"""
        return {"$bit": {field: {"and": value}}}

    @staticmethod
    def bit_or(field: str, value: int) -> Dict[str, Any]:
        """Bitwise OR ($bit or)"""
        return {"$bit": {field: {"or": value}}}

    @staticmethod
    def bit_xor(field: str, value: int) -> Dict[str, Any]:
        """Bitwise XOR ($bit xor)"""
        return {"$bit": {field: {"xor": value}}}

    @staticmethod
    def combine(*updates: Dict[str, Any]) -> Dict[str, Any]:
        """
        Combine multiple update operations into a single update document.

        Args:
            *updates: Update operation dictionaries

        Returns:
            Combined update document

        Example:
            >>> u = UpdateBuilder()
            >>> update = u.combine(
            ...     u.set('name', 'Alice'),
            ...     u.inc('views', 1),
            ...     u.push('tags', 'python')
            ... )
        """
        result = {}
        for update in updates:
            for operator, fields in update.items():
                if operator not in result:
                    result[operator] = {}
                result[operator].update(fields)
        return result
