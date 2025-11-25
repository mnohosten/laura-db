"""
LauraDB Aggregation - Aggregation pipeline builder
"""
from typing import Any, Dict, List, Union


class Aggregation:
    """
    Aggregation pipeline builder for constructing aggregation stages.

    This class provides methods for building aggregation pipelines with
    various stages like $match, $group, $project, $sort, $limit, and $skip.

    Example:
        >>> agg = Aggregation()
        >>> pipeline = [
        ...     agg.match({'age': {'$gte': 18}}),
        ...     agg.group('$city', {
        ...         'avgAge': agg.avg('$age'),
        ...         'count': agg.count()
        ...     }),
        ...     agg.sort({'avgAge': -1}),
        ...     agg.limit(10)
        ... ]
    """

    # Pipeline stages
    @staticmethod
    def match(filter: Dict[str, Any]) -> Dict[str, Any]:
        """
        Filter documents ($match stage).

        Args:
            filter: Query filter

        Returns:
            Match stage

        Example:
            >>> agg.match({'age': {'$gte': 18}})
        """
        return {"$match": filter}

    @staticmethod
    def group(
        group_by: Union[str, Dict[str, Any]],
        accumulators: Dict[str, Dict[str, Any]]
    ) -> Dict[str, Any]:
        """
        Group documents by key with aggregation operators ($group stage).

        Args:
            group_by: Field to group by (use '$fieldName') or compound key
            accumulators: Dictionary of accumulator expressions

        Returns:
            Group stage

        Example:
            >>> agg.group('$city', {
            ...     'avgAge': agg.avg('$age'),
            ...     'total': agg.sum('$amount'),
            ...     'count': agg.count()
            ... })
        """
        stage = {"_id": group_by}
        stage.update(accumulators)
        return {"$group": stage}

    @staticmethod
    def project(fields: Dict[str, Union[int, str, Dict]]) -> Dict[str, Any]:
        """
        Include, exclude, or transform fields ($project stage).

        Args:
            fields: Field specification (1 = include, 0 = exclude, or expression)

        Returns:
            Project stage

        Example:
            >>> agg.project({
            ...     'name': 1,
            ...     'age': 1,
            ...     'fullName': {'$concat': ['$firstName', ' ', '$lastName']},
            ...     '_id': 0
            ... })
        """
        return {"$project": fields}

    @staticmethod
    def sort(sort_spec: Dict[str, int]) -> Dict[str, Any]:
        """
        Sort documents ($sort stage).

        Args:
            sort_spec: Sort specification (1 = ascending, -1 = descending)

        Returns:
            Sort stage

        Example:
            >>> agg.sort({'age': -1, 'name': 1})
        """
        return {"$sort": sort_spec}

    @staticmethod
    def limit(count: int) -> Dict[str, Any]:
        """
        Limit number of documents ($limit stage).

        Args:
            count: Maximum number of documents

        Returns:
            Limit stage

        Example:
            >>> agg.limit(10)
        """
        return {"$limit": count}

    @staticmethod
    def skip(count: int) -> Dict[str, Any]:
        """
        Skip documents ($skip stage).

        Args:
            count: Number of documents to skip

        Returns:
            Skip stage

        Example:
            >>> agg.skip(20)
        """
        return {"$skip": count}

    @staticmethod
    def unwind(field: str, preserve_null: bool = False) -> Dict[str, Any]:
        """
        Deconstruct array field into separate documents ($unwind stage).

        Args:
            field: Array field to unwind (use '$fieldName')
            preserve_null: Preserve documents without the field

        Returns:
            Unwind stage

        Example:
            >>> agg.unwind('$tags')
        """
        if preserve_null:
            return {
                "$unwind": {
                    "path": field,
                    "preserveNullAndEmptyArrays": True
                }
            }
        return {"$unwind": field}

    @staticmethod
    def lookup(
        from_collection: str,
        local_field: str,
        foreign_field: str,
        as_field: str
    ) -> Dict[str, Any]:
        """
        Left outer join with another collection ($lookup stage).

        Args:
            from_collection: Collection to join with
            local_field: Field from input documents
            foreign_field: Field from documents of 'from' collection
            as_field: Output array field name

        Returns:
            Lookup stage

        Example:
            >>> agg.lookup('orders', 'userId', '_id', 'userOrders')
        """
        return {
            "$lookup": {
                "from": from_collection,
                "localField": local_field,
                "foreignField": foreign_field,
                "as": as_field
            }
        }

    # Aggregation operators (for use in $group)
    @staticmethod
    def sum(expression: Union[str, int]) -> Dict[str, Any]:
        """
        Sum values ($sum).

        Args:
            expression: Field to sum (use '$fieldName') or constant

        Example:
            >>> agg.sum('$amount')
            >>> agg.sum(1)  # Count
        """
        return {"$sum": expression}

    @staticmethod
    def avg(field: str) -> Dict[str, Any]:
        """
        Calculate average ($avg).

        Args:
            field: Field to average (use '$fieldName')

        Example:
            >>> agg.avg('$age')
        """
        return {"$avg": field}

    @staticmethod
    def min_(field: str) -> Dict[str, Any]:
        """
        Find minimum value ($min).

        Args:
            field: Field to find minimum (use '$fieldName')

        Example:
            >>> agg.min_('$price')
        """
        return {"$min": field}

    @staticmethod
    def max_(field: str) -> Dict[str, Any]:
        """
        Find maximum value ($max).

        Args:
            field: Field to find maximum (use '$fieldName')

        Example:
            >>> agg.max_('$price')
        """
        return {"$max": field}

    @staticmethod
    def count() -> Dict[str, Any]:
        """
        Count documents ($count).

        Example:
            >>> agg.count()
        """
        return {"$count": {}}

    @staticmethod
    def push(field: str) -> Dict[str, Any]:
        """
        Create array of values ($push).

        Args:
            field: Field to collect (use '$fieldName')

        Example:
            >>> agg.push('$name')
        """
        return {"$push": field}

    @staticmethod
    def add_to_set(field: str) -> Dict[str, Any]:
        """
        Create array of unique values ($addToSet).

        Args:
            field: Field to collect (use '$fieldName')

        Example:
            >>> agg.add_to_set('$category')
        """
        return {"$addToSet": field}

    @staticmethod
    def first(field: str) -> Dict[str, Any]:
        """
        Get first value ($first).

        Args:
            field: Field to get (use '$fieldName')

        Example:
            >>> agg.first('$createdAt')
        """
        return {"$first": field}

    @staticmethod
    def last(field: str) -> Dict[str, Any]:
        """
        Get last value ($last).

        Args:
            field: Field to get (use '$fieldName')

        Example:
            >>> agg.last('$updatedAt')
        """
        return {"$last": field}

    # Expression operators (for use in $project)
    @staticmethod
    def concat(*fields: str) -> Dict[str, Any]:
        """
        Concatenate strings ($concat).

        Args:
            *fields: Fields or strings to concatenate

        Example:
            >>> agg.concat('$firstName', ' ', '$lastName')
        """
        return {"$concat": list(fields)}

    @staticmethod
    def substring(field: str, start: int, length: int) -> Dict[str, Any]:
        """
        Extract substring ($substr).

        Args:
            field: String field (use '$fieldName')
            start: Start position
            length: Length of substring

        Example:
            >>> agg.substring('$name', 0, 5)
        """
        return {"$substr": [field, start, length]}

    @staticmethod
    def to_upper(field: str) -> Dict[str, Any]:
        """
        Convert to uppercase ($toUpper).

        Args:
            field: String field (use '$fieldName')

        Example:
            >>> agg.to_upper('$name')
        """
        return {"$toUpper": field}

    @staticmethod
    def to_lower(field: str) -> Dict[str, Any]:
        """
        Convert to lowercase ($toLower).

        Args:
            field: String field (use '$fieldName')

        Example:
            >>> agg.to_lower('$email')
        """
        return {"$toLower": field}

    @staticmethod
    def cond(condition: Dict[str, Any], true_expr: Any, false_expr: Any) -> Dict[str, Any]:
        """
        Conditional expression ($cond).

        Args:
            condition: Condition expression
            true_expr: Value if condition is true
            false_expr: Value if condition is false

        Example:
            >>> agg.cond(
            ...     {'$gte': ['$age', 18]},
            ...     'adult',
            ...     'minor'
            ... )
        """
        return {"$cond": [condition, true_expr, false_expr]}
