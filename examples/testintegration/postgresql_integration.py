#!/usr/bin/env python3
"""
PostgreSQL Integration for SecAuto
Provides database connectivity and query capabilities for automations
"""

import json
import sys
import logging
import warnings
from datetime import datetime, date
from decimal import Decimal
from typing import Dict, Any, List, Optional

# Suppress all warnings including SSL/urllib3 compatibility warnings
import os
os.environ['PYTHONWARNINGS'] = 'ignore'
warnings.filterwarnings("ignore")

# Set environment variable to suppress urllib3 warnings before any imports
os.environ['URLLIB3_DISABLE_WARNINGS'] = '1'

import psycopg2
from psycopg2 import sql, extras
from psycopg2.pool import SimpleConnectionPool
from contextlib import contextmanager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class DateTimeEncoder(json.JSONEncoder):
    """Custom JSON encoder to handle datetime and decimal objects"""
    def default(self, obj):
        if isinstance(obj, (datetime, date)):
            return obj.isoformat()
        elif isinstance(obj, Decimal):
            return float(obj)
        return super().default(obj)

class PostgreSQLIntegration:
    """PostgreSQL database integration"""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize PostgreSQL integration
        
        Args:
            config: Configuration dictionary with connection parameters
        """
        self.config = config
        self.client_id = config.get('_client_id')
        self.pool = None
        self._initialize_pool()
    
    def _initialize_pool(self):
        """Initialize connection pool"""
        try:
            conn_params = {
                'host': self.config.get('host', 'localhost'),
                'port': self.config.get('port', 5432),
                'database': self.config.get('database'),
                'user': self.config.get('username'),
                'password': self.config.get('password'),
                'sslmode': self.config.get('ssl_mode', 'prefer')
            }
            
            # Create connection pool (min 1, max 5 connections)
            self.pool = SimpleConnectionPool(1, 5, **conn_params)
            logger.info(f"PostgreSQL connection pool initialized for client: {self.client_id or 'default'}")
        except Exception as e:
            logger.error(f"Failed to initialize connection pool: {e}")
            raise
    
    @contextmanager
    def get_connection(self):
        """Get a connection from the pool"""
        conn = None
        try:
            conn = self.pool.getconn()
            yield conn
            conn.commit()
        except Exception as e:
            if conn:
                conn.rollback()
            raise e
        finally:
            if conn:
                self.pool.putconn(conn)
    
    def test_connection(self) -> Dict[str, Any]:
        """Test the database connection"""
        try:
            with self.get_connection() as conn:
                with conn.cursor() as cursor:
                    cursor.execute("SELECT version()")
                    version = cursor.fetchone()[0]
                    
                    cursor.execute("SELECT current_database()")
                    database = cursor.fetchone()[0]
                    
                    return {
                        'success': True,
                        'connected': True,
                        'database': database,
                        'version': version
                    }
        except Exception as e:
            return {
                'success': False,
                'connected': False,
                'error': str(e)
            }
    
    def list_tables(self, schema: str = 'public') -> Dict[str, Any]:
        """
        List all tables in the database
        
        Args:
            schema: Schema name (default: public)
        
        Returns:
            Dictionary with table information
        """
        try:
            with self.get_connection() as conn:
                with conn.cursor(cursor_factory=extras.RealDictCursor) as cursor:
                    query = """
                    SELECT 
                        table_name,
                        table_type,
                        (SELECT COUNT(*) 
                         FROM information_schema.columns 
                         WHERE table_schema = t.table_schema 
                         AND table_name = t.table_name) as column_count
                    FROM information_schema.tables t
                    WHERE table_schema = %s
                    ORDER BY table_name
                    """
                    cursor.execute(query, (schema,))
                    tables = cursor.fetchall()
                    
                    return {
                        'success': True,
                        'schema': schema,
                        'table_count': len(tables),
                        'tables': [dict(t) for t in tables]
                    }
        except Exception as e:
            logger.error(f"Error listing tables: {e}")
            return {
                'success': False,
                'error': str(e)
            }
    
    def list_items(self, table: str, limit: int = 100, offset: int = 0, 
                   order_by: str = None, filters: Dict = None) -> Dict[str, Any]:
        """
        List items from a specific table
        
        Args:
            table: Table name to query
            limit: Maximum number of items to return
            offset: Number of items to skip
            order_by: Column to order by
            filters: Column filters as key-value pairs
        
        Returns:
            Dictionary with items from the table
        """
        try:
            # Validate table name to prevent SQL injection
            if not self._validate_table_name(table):
                return {
                    'success': False,
                    'error': 'Invalid table name'
                }
            
            with self.get_connection() as conn:
                with conn.cursor(cursor_factory=extras.RealDictCursor) as cursor:
                    # Build query - use f-string with validated table name (already validated above)
                    query_parts = [f"SELECT * FROM {table}"]
                    params = []
                    
                    # Add filters if provided
                    if filters:
                        where_clauses = []
                        for col, val in filters.items():
                            # Validate column name similar to table name
                            if not self._validate_table_name(col):
                                return {'success': False, 'error': f'Invalid column name: {col}'}
                            where_clauses.append(f"{col} = %s")
                            params.append(val)
                        if where_clauses:
                            query_parts.append(f"WHERE {' AND '.join(where_clauses)}")
                    
                    # Add ordering
                    if order_by:
                        if not self._validate_table_name(order_by):
                            return {'success': False, 'error': f'Invalid order_by column: {order_by}'}
                        query_parts.append(f"ORDER BY {order_by}")
                    
                    # Add limit and offset
                    query_parts.append("LIMIT %s OFFSET %s")
                    params.extend([limit, offset])
                    
                    query = " ".join(query_parts)
                    cursor.execute(query, params)
                    items = cursor.fetchall()
                    
                    # Get total count
                    count_query = f"SELECT COUNT(*) FROM {table}"
                    if filters:
                        where_clauses = []
                        count_params = []
                        for col, val in filters.items():
                            where_clauses.append(f"{col} = %s")
                            count_params.append(val)
                        if where_clauses:
                            count_query += f" WHERE {' AND '.join(where_clauses)}"
                            cursor.execute(count_query, count_params)
                    else:
                        cursor.execute(count_query)
                    
                    count_result = cursor.fetchone()
                    # With RealDictCursor, the result is a dict with column name as key
                    total_count = count_result['count']
                    
                    items_dict = [dict(item) for item in items]
                    
                    return {
                        'success': True,
                        'table': table,
                        'total_count': total_count,
                        'returned_count': len(items),
                        'limit': limit,
                        'offset': offset,
                        'items': items_dict
                    }
        except Exception as e:
            logger.error(f"Error listing items from {table}: {e}", exc_info=True)
            return {
                'success': False,
                'error': str(e),
                'error_type': type(e).__name__
            }
    
    def query(self, sql_query: str, params: List = None) -> Dict[str, Any]:
        """
        Execute a custom SQL query (SELECT only for safety)
        
        Args:
            sql_query: SQL query to execute
            params: Query parameters for parameterized queries
        
        Returns:
            Query results
        """
        try:
            # Basic safety check - only allow SELECT queries
            if not sql_query.strip().upper().startswith('SELECT'):
                return {
                    'success': False,
                    'error': 'Only SELECT queries are allowed for safety'
                }
            
            with self.get_connection() as conn:
                with conn.cursor(cursor_factory=extras.RealDictCursor) as cursor:
                    if params:
                        cursor.execute(sql_query, params)
                    else:
                        cursor.execute(sql_query)
                    
                    results = cursor.fetchall()
                    
                    return {
                        'success': True,
                        'row_count': len(results),
                        'results': [dict(row) for row in results]
                    }
        except Exception as e:
            logger.error(f"Error executing query: {e}")
            return {
                'success': False,
                'error': str(e)
            }
    
    def get_table_info(self, table: str) -> Dict[str, Any]:
        """
        Get detailed information about a table
        
        Args:
            table: Table name
        
        Returns:
            Table schema information
        """
        try:
            if not self._validate_table_name(table):
                return {
                    'success': False,
                    'error': 'Invalid table name'
                }
            
            with self.get_connection() as conn:
                with conn.cursor(cursor_factory=extras.RealDictCursor) as cursor:
                    # Get column information
                    query = """
                    SELECT 
                        column_name,
                        data_type,
                        is_nullable,
                        column_default,
                        character_maximum_length
                    FROM information_schema.columns
                    WHERE table_name = %s
                    ORDER BY ordinal_position
                    """
                    cursor.execute(query, (table,))
                    columns = cursor.fetchall()
                    
                    # Get primary key information
                    pk_query = """
                    SELECT a.attname
                    FROM pg_index i
                    JOIN pg_attribute a ON a.attrelid = i.indrelid
                        AND a.attnum = ANY(i.indkey)
                    WHERE i.indrelid = %s::regclass
                        AND i.indisprimary
                    """
                    cursor.execute(pk_query, (table,))
                    primary_keys = [row['attname'] for row in cursor.fetchall()]
                    
                    # Get row count
                    cursor.execute(f"SELECT COUNT(*) FROM {table}")
                    row_count_result = cursor.fetchone()
                    # With RealDictCursor, the result is a dict with column name as key
                    row_count = row_count_result['count']
                    
                    return {
                        'success': True,
                        'table': table,
                        'row_count': row_count,
                        'column_count': len(columns),
                        'primary_keys': primary_keys,
                        'columns': [dict(col) for col in columns]
                    }
        except Exception as e:
            logger.error(f"Error getting table info for {table}: {e}", exc_info=True)
            return {
                'success': False,
                'error': str(e),
                'error_type': type(e).__name__
            }
    
    def _validate_table_name(self, table: str) -> bool:
        """Validate table name to prevent SQL injection"""
        # Basic validation - alphanumeric, underscore, and dot (for schema.table)
        import re
        return bool(re.match(r'^[a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+)?$', table))
    
    def cleanup(self):
        """Clean up resources"""
        if self.pool:
            self.pool.closeall()
            logger.info("Connection pool closed")

# Global integration instance
_integration = None

def initialize_integration(context: Dict[str, Any]) -> PostgreSQLIntegration:
    """Initialize the integration with context from SecAuto"""
    global _integration
    
    # Extract client ID and configuration from context
    client_id = context.get('client_id')
    config = context.get('config', {})
    #credentials = context.get('credentials', {})
    
    # Log client context
    if client_id:
        logger.info(f"Initializing PostgreSQL integration for client: {client_id}")
    
    # Merge credentials into config
    #config.update(credentials)
    
    # Store client_id in config for reference
    config['_client_id'] = client_id
    
    _integration = PostgreSQLIntegration(config)
    return _integration

# Integration function wrappers for SecAuto
def list_tables(schema: str = 'public') -> Dict[str, Any]:
    """List all tables in the database"""
    if not _integration:
        return {'success': False, 'error': 'Integration not initialized'}
    return _integration.list_tables(schema)

def list_items(table: str, limit: int = 100, offset: int = 0, 
               order_by: str = None, filters: Dict = None) -> Dict[str, Any]:
    """List items from a specific table"""
    if not _integration:
        return {'success': False, 'error': 'Integration not initialized'}
    return _integration.list_items(table, limit, offset, order_by, filters)

def query(sql: str, params: List = None) -> Dict[str, Any]:
    """Execute a custom SQL query"""
    if not _integration:
        return {'success': False, 'error': 'Integration not initialized'}
    return _integration.query(sql, params)

def get_table_info(table: str) -> Dict[str, Any]:
    """Get detailed information about a table"""
    if not _integration:
        return {'success': False, 'error': 'Integration not initialized'}
    return _integration.get_table_info(table)

def test_connection() -> Dict[str, Any]:
    """Test the database connection"""
    if not _integration:
        return {'success': False, 'error': 'Integration not initialized'}
    return _integration.test_connection()

# Main execution for SecAuto integration system
if __name__ == "__main__":
    try:
        # Read context from stdin
        context_json = sys.stdin.read().strip()
        
        if not context_json:
            error_result = {
                'success': False,
                'error': 'No context provided via stdin'
            }
            print(json.dumps(error_result, cls=DateTimeEncoder))
            sys.exit(1)
        
        context = json.loads(context_json)
        
        # Debug: Log the received context (remove in production)
        logger.info(f"Received context: {json.dumps(context, indent=2)}")
        
        # Initialize integration
        integration = initialize_integration(context)
        
        # Get function to execute
        function = context.get('function')
        params = context.get('params') or {}
        
        logger.info(f"Executing function: {function} with params: {params}")
        
        # Execute the requested function
        if function == 'list_tables':
            result = list_tables(**params)
        elif function == 'list_items':
            result = list_items(**params)
        elif function == 'query':
            result = query(**params)
        elif function == 'get_table_info':
            result = get_table_info(**params)
        elif function == 'test_connection':
            result = test_connection()
        else:
            result = {
                'success': False,
                'error': f'Unknown function: {function}. Available functions: list_tables, list_items, query, get_table_info, test_connection'
            }
        
        logger.info(f"Function result: {result}")
        
        # Output result as JSON
        print(json.dumps(result, cls=DateTimeEncoder))
        
        # Cleanup
        if integration:
            integration.cleanup()
        
    except json.JSONDecodeError as e:
        error_result = {
            'success': False,
            'error': f'Invalid JSON context: {str(e)}'
        }
        print(json.dumps(error_result, cls=DateTimeEncoder))
        sys.exit(1)
    except Exception as e:
        logger.error(f"Integration error: {str(e)}", exc_info=True)
        error_result = {
            'success': False,
            'error': str(e),
            'error_type': type(e).__name__
        }
        print(json.dumps(error_result, cls=DateTimeEncoder))
        sys.exit(1)