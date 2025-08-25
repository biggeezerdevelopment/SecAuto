#!/usr/bin/env python3
"""
Database Monitor Automation
Monitors PostgreSQL database tables and triggers alerts based on conditions
"""

import json
import sys
import logging
from datetime import datetime
from typing import Dict, Any, List

# Import SecAuto base functions
from secauto_base import load_context, return_context, get_client_id, use_integration

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class DatabaseMonitor:
    """Database monitoring automation"""
    
    def __init__(self, config: Dict[str, Any], client_id: str = None):
        """
        Initialize the database monitor
        
        Args:
            config: Configuration dictionary
            client_id: Client ID for multi-tenant support
        """
        self.config = config
        self.client_id = client_id
        self.integration_name = config.get('integration', 'postgresql')
        self.monitoring_rules = config.get('monitoring_rules', [])
        self.alert_config = config.get('alert_config', {})
        self.results = []
        self.alerts = []
    
    def run(self) -> Dict[str, Any]:
        """
        Run the monitoring automation
        
        Returns:
            Dictionary with monitoring results
        """
        try:
            # Test connection first
            connection_result = use_integration(
                self.integration_name, 
                'test_connection',
                client_id=self.client_id
            )
            
            if not connection_result.get('success'):
                return {
                    'success': False,
                    'error': 'Database connection failed',
                    'details': connection_result
                }
            
            # Process each monitoring rule
            for rule in self.monitoring_rules:
                self.process_rule(rule)
            
            # Generate summary
            summary = self.generate_summary()
            
            # Send alerts if needed
            if self.alerts:
                self.send_alerts()
            
            return {
                'success': True,
                'timestamp': datetime.utcnow().isoformat(),
                'connection': connection_result.get('result', {}),
                'rules_processed': len(self.monitoring_rules),
                'alerts_triggered': len(self.alerts),
                'summary': summary,
                'results': self.results,
                'alerts': self.alerts
            }
            
        except Exception as e:
            logger.error(f"Monitor error: {e}")
            return {
                'success': False,
                'error': str(e)
            }
    
    def process_rule(self, rule: Dict[str, Any]):
        """
        Process a single monitoring rule
        
        Args:
            rule: Monitoring rule configuration
        """
        rule_name = rule.get('name', 'Unnamed Rule')
        rule_type = rule.get('type', 'row_count')
        
        try:
            if rule_type == 'row_count':
                self.check_row_count(rule)
            elif rule_type == 'table_changes':
                self.check_table_changes(rule)
            elif rule_type == 'query_result':
                self.check_query_result(rule)
            elif rule_type == 'table_list':
                self.check_table_list(rule)
            else:
                logger.warning(f"Unknown rule type: {rule_type}")
                
        except Exception as e:
            logger.error(f"Error processing rule '{rule_name}': {e}")
            self.results.append({
                'rule': rule_name,
                'success': False,
                'error': str(e)
            })
    
    def check_row_count(self, rule: Dict[str, Any]):
        """Check row count threshold for a table"""
        table = rule.get('table')
        threshold = rule.get('threshold', 1000)
        comparison = rule.get('comparison', 'greater_than')
        
        # Get items from table
        result = use_integration(
            self.integration_name,
            'list_items',
            client_id=self.client_id,
            table=table,
            limit=1  # Just need count
        )
        
        if result.get('success'):
            data = result.get('result', {})
            total_count = data.get('total_count', 0)
            
            # Check threshold
            alert_triggered = False
            if comparison == 'greater_than' and total_count > threshold:
                alert_triggered = True
            elif comparison == 'less_than' and total_count < threshold:
                alert_triggered = True
            elif comparison == 'equals' and total_count == threshold:
                alert_triggered = True
            
            self.results.append({
                'rule': rule.get('name'),
                'type': 'row_count',
                'table': table,
                'count': total_count,
                'threshold': threshold,
                'comparison': comparison,
                'alert_triggered': alert_triggered
            })
            
            if alert_triggered:
                self.alerts.append({
                    'rule': rule.get('name'),
                    'severity': rule.get('severity', 'medium'),
                    'message': f"Table '{table}' has {total_count} rows ({comparison} {threshold})",
                    'details': {
                        'table': table,
                        'count': total_count,
                        'threshold': threshold
                    }
                })
    
    def check_table_changes(self, rule: Dict[str, Any]):
        """Check for recent changes in a table"""
        table = rule.get('table')
        time_column = rule.get('time_column', 'updated_at')
        minutes = rule.get('minutes', 60)
        
        # Query for recent changes
        sql_query = f"""
        SELECT COUNT(*) as change_count
        FROM {table}
        WHERE {time_column} >= NOW() - INTERVAL '{minutes} minutes'
        """
        
        result = use_integration(
            self.integration_name,
            'query',
            client_id=self.client_id,
            sql=sql_query
        )
        
        if result.get('success'):
            data = result.get('result', {})
            results = data.get('results', [])
            change_count = results[0].get('change_count', 0) if results else 0
            
            alert_triggered = change_count > rule.get('threshold', 0)
            
            self.results.append({
                'rule': rule.get('name'),
                'type': 'table_changes',
                'table': table,
                'changes': change_count,
                'period_minutes': minutes,
                'alert_triggered': alert_triggered
            })
            
            if alert_triggered:
                self.alerts.append({
                    'rule': rule.get('name'),
                    'severity': rule.get('severity', 'high'),
                    'message': f"Table '{table}' has {change_count} changes in last {minutes} minutes",
                    'details': {
                        'table': table,
                        'changes': change_count,
                        'period': f"{minutes} minutes"
                    }
                })
    
    def check_query_result(self, rule: Dict[str, Any]):
        """Execute a custom query and check results"""
        query = rule.get('query')
        expected_count = rule.get('expected_count')
        
        result = use_integration(
            self.integration_name,
            'query',
            client_id=self.client_id,
            sql=query
        )
        
        if result.get('success'):
            data = result.get('result', {})
            row_count = data.get('row_count', 0)
            
            alert_triggered = False
            if expected_count is not None:
                alert_triggered = row_count != expected_count
            elif rule.get('min_count') is not None:
                alert_triggered = row_count < rule.get('min_count')
            elif rule.get('max_count') is not None:
                alert_triggered = row_count > rule.get('max_count')
            
            self.results.append({
                'rule': rule.get('name'),
                'type': 'query_result',
                'row_count': row_count,
                'alert_triggered': alert_triggered
            })
            
            if alert_triggered:
                self.alerts.append({
                    'rule': rule.get('name'),
                    'severity': rule.get('severity', 'medium'),
                    'message': f"Query returned {row_count} rows",
                    'details': {
                        'query': query[:100] + '...' if len(query) > 100 else query,
                        'row_count': row_count
                    }
                })
    
    def check_table_list(self, rule: Dict[str, Any]):
        """Check for expected tables in database"""
        expected_tables = rule.get('expected_tables', [])
        schema = rule.get('schema', 'public')
        
        result = use_integration(
            self.integration_name,
            'list_tables',
            client_id=self.client_id,
            schema=schema
        )
        
        if result.get('success'):
            data = result.get('result', {})
            tables = data.get('tables', [])
            table_names = [t.get('table_name') for t in tables]
            
            missing_tables = [t for t in expected_tables if t not in table_names]
            
            alert_triggered = len(missing_tables) > 0
            
            self.results.append({
                'rule': rule.get('name'),
                'type': 'table_list',
                'schema': schema,
                'found_tables': len(table_names),
                'missing_tables': missing_tables,
                'alert_triggered': alert_triggered
            })
            
            if alert_triggered:
                self.alerts.append({
                    'rule': rule.get('name'),
                    'severity': rule.get('severity', 'critical'),
                    'message': f"Missing {len(missing_tables)} expected tables",
                    'details': {
                        'schema': schema,
                        'missing_tables': missing_tables
                    }
                })
    
    def generate_summary(self) -> Dict[str, Any]:
        """Generate monitoring summary"""
        return {
            'total_rules': len(self.monitoring_rules),
            'successful_checks': len([r for r in self.results if r.get('success', True)]),
            'failed_checks': len([r for r in self.results if not r.get('success', True)]),
            'alerts_triggered': len(self.alerts),
            'alert_severity_breakdown': {
                'critical': len([a for a in self.alerts if a['severity'] == 'critical']),
                'high': len([a for a in self.alerts if a['severity'] == 'high']),
                'medium': len([a for a in self.alerts if a['severity'] == 'medium']),
                'low': len([a for a in self.alerts if a['severity'] == 'low'])
            }
        }
    
    def send_alerts(self):
        """Send alerts (placeholder for actual alerting logic)"""
        # This would integrate with your alerting system
        # For now, just log the alerts
        for alert in self.alerts:
            logger.warning(f"ALERT [{alert['severity'].upper()}]: {alert['message']}")

def main():
    """Main execution function"""
    # Load context from SecAuto
    context = load_context()
    
    # Extract client ID from context
    client_id = context.get('client_id')
    
    # Log client context
    if client_id:
        logger.info(f"Running database monitor for client: {client_id}")
    
    # Default configuration
    default_config = {
        'integration': 'postgresql',
        'monitoring_rules': [
            {
                'name': 'Check users table',
                'type': 'row_count',
                'table': 'users',
                'threshold': 0,
                'comparison': 'equals',
                'severity': 'high'
            }
        ],
        'alert_config': {
            'enabled': True,
            'channels': ['log', 'webhook']
        }
    }
    
    # Merge with provided configuration
    config = context.get('config', {})
    for key, value in default_config.items():
        if key not in config:
            config[key] = value
    
    # Create and run monitor
    monitor = DatabaseMonitor(config, client_id)
    result = monitor.run()
    
    # Return results to SecAuto
    return_context(result)

if __name__ == "__main__":
    main()