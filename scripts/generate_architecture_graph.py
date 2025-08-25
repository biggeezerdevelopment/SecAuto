#!/usr/bin/env python3
"""
SecAuto Architecture Visualization
Generates a visual graph showing the code flow and component relationships
"""

import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch
import networkx as nx
from networkx.drawing.nx_agraph import graphviz_layout
import warnings
warnings.filterwarnings('ignore')

# Create a directed graph
G = nx.DiGraph()

# Define node categories with colors
node_colors = {
    'core': '#4A90E2',        # Blue - Core server components
    'manager': '#50C878',     # Green - Manager components
    'service': '#FF6B6B',     # Red - Service layer
    'storage': '#FFD700',     # Gold - Storage/persistence
    'api': '#9B59B6',         # Purple - API/endpoints
    'security': '#E74C3C',    # Dark Red - Security components
    'external': '#95A5A6',    # Gray - External services
    'integration': '#FF9F40', # Orange - Integration layer
}

# Add nodes with categories
nodes = {
    # Core Server
    'SecAutoServer': 'core',
    'main.go': 'core',
    
    # Configuration & Logging
    'Config': 'service',
    'Logger': 'service',
    'Validator': 'service',
    
    # Manager Components
    'JobManager': 'manager',
    'IntegrationManager': 'manager',
    'ClientIntegrationManager': 'manager',
    'AutomationManager': 'manager',
    'ClusterManager': 'manager',
    'ScheduleManager': 'manager',
    'PlaybookManager': 'manager',
    'ClientManager': 'manager',
    
    # Security & Auth
    'APIKeyManager': 'security',
    'SecurityMiddleware': 'security',
    'AuditLogger': 'security',
    'TLSManager': 'security',
    'RateLimiter': 'security',
    
    # Storage & Cache
    'Redis': 'storage',
    'JobStore': 'storage',
    'Cache': 'storage',
    
    # Services
    'RulesEngine': 'service',
    'SwaggerUI': 'service',
    
    # API Endpoints
    'Health API': 'api',
    'Playbook API': 'api',
    'Jobs API': 'api',
    'Integration API': 'api',
    'Cache API': 'api',
    'Cluster API': 'api',
    'Schedule API': 'api',
    'Client API': 'api',
    'Automation API': 'api',
    
    # External Services
    'Python Scripts': 'external',
    'SoarBaseAPI.py': 'external',
    'integration_loader.py': 'external',
    
    # Integration Layer
    'HTTP Server': 'integration',
    'Middleware Chain': 'integration',
}

# Add nodes to graph
for node, category in nodes.items():
    G.add_node(node, category=category)

# Define edges (dependencies and data flow)
edges = [
    # Main initialization flow
    ('main.go', 'SecAutoServer'),
    ('SecAutoServer', 'Config'),
    ('Config', 'Logger'),
    ('SecAutoServer', 'Validator'),
    ('SecAutoServer', 'RulesEngine'),
    
    # Storage initialization
    ('SecAutoServer', 'Redis'),
    ('Redis', 'JobStore'),
    ('Redis', 'Cache'),
    
    # Manager initialization
    ('SecAutoServer', 'JobManager'),
    ('JobManager', 'JobStore'),
    ('SecAutoServer', 'IntegrationManager'),
    ('SecAutoServer', 'ClientIntegrationManager'),
    ('SecAutoServer', 'AutomationManager'),
    ('SecAutoServer', 'ClusterManager'),
    ('SecAutoServer', 'ScheduleManager'),
    ('SecAutoServer', 'PlaybookManager'),
    ('SecAutoServer', 'ClientManager'),
    
    # Security components
    ('SecAutoServer', 'APIKeyManager'),
    ('SecAutoServer', 'SecurityMiddleware'),
    ('SecAutoServer', 'AuditLogger'),
    ('SecAutoServer', 'TLSManager'),
    ('SecurityMiddleware', 'RateLimiter'),
    
    # HTTP Server setup
    ('SecAutoServer', 'HTTP Server'),
    ('HTTP Server', 'Middleware Chain'),
    ('Middleware Chain', 'SecurityMiddleware'),
    
    # API endpoint connections
    ('Middleware Chain', 'Health API'),
    ('Middleware Chain', 'Playbook API'),
    ('Middleware Chain', 'Jobs API'),
    ('Middleware Chain', 'Integration API'),
    ('Middleware Chain', 'Cache API'),
    ('Middleware Chain', 'Cluster API'),
    ('Middleware Chain', 'Schedule API'),
    ('Middleware Chain', 'Client API'),
    ('Middleware Chain', 'Automation API'),
    
    # API to Manager connections
    ('Playbook API', 'PlaybookManager'),
    ('Jobs API', 'JobManager'),
    ('Integration API', 'IntegrationManager'),
    ('Cache API', 'Cache'),
    ('Cluster API', 'ClusterManager'),
    ('Schedule API', 'ScheduleManager'),
    ('Client API', 'ClientManager'),
    ('Automation API', 'AutomationManager'),
    
    # Manager to storage connections
    ('PlaybookManager', 'Redis'),
    ('JobManager', 'Redis'),
    ('IntegrationManager', 'Redis'),
    ('ClusterManager', 'Redis'),
    ('ScheduleManager', 'Redis'),
    ('ClientManager', 'Redis'),
    ('AutomationManager', 'Redis'),
    
    # Integration with Python components
    ('IntegrationManager', 'Python Scripts'),
    ('IntegrationManager', 'integration_loader.py'),
    ('ClientIntegrationManager', 'SoarBaseAPI.py'),
    ('PlaybookManager', 'Python Scripts'),
    ('AutomationManager', 'Python Scripts'),
    
    # Swagger documentation
    ('HTTP Server', 'SwaggerUI'),
    
    # Cross-component dependencies
    ('PlaybookManager', 'RulesEngine'),
    ('JobManager', 'RulesEngine'),
    ('ScheduleManager', 'JobManager'),
    ('ClientIntegrationManager', 'IntegrationManager'),
]

# Add edges to graph
G.add_edges_from(edges)

# Create the plot with a larger figure
fig, ax = plt.subplots(1, 1, figsize=(24, 18))

# Try to use graphviz layout for better organization
try:
    pos = graphviz_layout(G, prog='dot', args='-Grankdir=TB')
except:
    # Fallback to spring layout if graphviz is not available
    pos = nx.spring_layout(G, k=3, iterations=50, seed=42)

# Draw nodes with different colors based on category
for category, color in node_colors.items():
    node_list = [node for node, cat in nodes.items() if cat == category]
    nx.draw_networkx_nodes(G, pos, 
                          nodelist=node_list,
                          node_color=color,
                          node_size=3000,
                          alpha=0.9,
                          ax=ax)

# Draw edges with different styles
# Main flow edges (thicker)
main_flow = [
    ('main.go', 'SecAutoServer'),
    ('SecAutoServer', 'HTTP Server'),
    ('HTTP Server', 'Middleware Chain'),
]
nx.draw_networkx_edges(G, pos, 
                       edgelist=main_flow,
                       width=3,
                       alpha=0.7,
                       edge_color='black',
                       arrows=True,
                       arrowsize=20,
                       ax=ax)

# Storage connections (dashed)
storage_edges = [(u, v) for u, v in edges if 'Redis' in v or 'Cache' in v or 'JobStore' in v]
nx.draw_networkx_edges(G, pos,
                       edgelist=storage_edges,
                       width=2,
                       alpha=0.5,
                       edge_color='orange',
                       style='dashed',
                       arrows=True,
                       arrowsize=15,
                       ax=ax)

# External integration edges (dotted)
external_edges = [(u, v) for u, v in edges if 'Python' in v or '.py' in v]
nx.draw_networkx_edges(G, pos,
                       edgelist=external_edges,
                       width=2,
                       alpha=0.5,
                       edge_color='purple',
                       style='dotted',
                       arrows=True,
                       arrowsize=15,
                       ax=ax)

# Regular edges
regular_edges = [e for e in edges if e not in main_flow and e not in storage_edges and e not in external_edges]
nx.draw_networkx_edges(G, pos,
                       edgelist=regular_edges,
                       width=1.5,
                       alpha=0.5,
                       edge_color='gray',
                       arrows=True,
                       arrowsize=15,
                       ax=ax)

# Draw labels
nx.draw_networkx_labels(G, pos, 
                        font_size=8,
                        font_weight='bold',
                        ax=ax)

# Create legend
legend_elements = [
    mpatches.Patch(color=node_colors['core'], label='Core Server'),
    mpatches.Patch(color=node_colors['manager'], label='Managers'),
    mpatches.Patch(color=node_colors['service'], label='Services'),
    mpatches.Patch(color=node_colors['storage'], label='Storage/Cache'),
    mpatches.Patch(color=node_colors['api'], label='API Endpoints'),
    mpatches.Patch(color=node_colors['security'], label='Security'),
    mpatches.Patch(color=node_colors['external'], label='External/Python'),
    mpatches.Patch(color=node_colors['integration'], label='Integration Layer'),
]

ax.legend(handles=legend_elements, loc='upper left', fontsize=10)

# Add title and annotations
plt.title('SecAuto Architecture and Code Flow', fontsize=20, fontweight='bold', pad=20)

# Add text box with key information
textstr = '\n'.join([
    'Key Components:',
    '• Go-based core server with modular architecture',
    '• Redis for distributed storage and caching',
    '• Python integration layer for custom scripts',
    '• RESTful API with middleware chain',
    '• Manager pattern for component isolation',
    '• Security middleware with auth & rate limiting'
])

props = dict(boxstyle='round', facecolor='wheat', alpha=0.5)
ax.text(0.02, 0.98, textstr, transform=ax.transAxes, fontsize=10,
        verticalalignment='top', bbox=props)

# Remove axis
ax.axis('off')

# Adjust layout
plt.tight_layout()

# Save the figure
output_file = '/Volumes/My Shared Files/Home/Downloads/SecAuto/secauto_architecture_graph.png'
plt.savefig(output_file, dpi=300, bbox_inches='tight', facecolor='white')
print(f"Architecture graph saved to: {output_file}")

# Also create a simplified data flow diagram
fig2, ax2 = plt.subplots(1, 1, figsize=(16, 10))

# Simplified flow graph
flow_G = nx.DiGraph()

# Main data flow nodes
flow_nodes = {
    'Client Request': 'external',
    'HTTP Server': 'integration',
    'Security Middleware': 'security',
    'API Router': 'api',
    'Business Logic': 'manager',
    'Redis Storage': 'storage',
    'Python Integration': 'external',
    'Response': 'external',
}

# Add flow nodes
for node, category in flow_nodes.items():
    flow_G.add_node(node, category=category)

# Define flow edges
flow_edges = [
    ('Client Request', 'HTTP Server'),
    ('HTTP Server', 'Security Middleware'),
    ('Security Middleware', 'API Router'),
    ('API Router', 'Business Logic'),
    ('Business Logic', 'Redis Storage'),
    ('Business Logic', 'Python Integration'),
    ('Redis Storage', 'Business Logic'),
    ('Python Integration', 'Business Logic'),
    ('Business Logic', 'Response'),
    ('Response', 'Client Request'),
]

flow_G.add_edges_from(flow_edges)

# Layout for flow diagram
pos_flow = nx.circular_layout(flow_G)

# Draw flow diagram
for category, color in node_colors.items():
    node_list = [node for node, cat in flow_nodes.items() if cat == category]
    if node_list:
        nx.draw_networkx_nodes(flow_G, pos_flow,
                              nodelist=node_list,
                              node_color=color,
                              node_size=4000,
                              alpha=0.9,
                              ax=ax2)

nx.draw_networkx_edges(flow_G, pos_flow,
                       width=3,
                       alpha=0.7,
                       edge_color='darkblue',
                       arrows=True,
                       arrowsize=25,
                       connectionstyle='arc3,rad=0.1',
                       ax=ax2)

nx.draw_networkx_labels(flow_G, pos_flow,
                        font_size=11,
                        font_weight='bold',
                        ax=ax2)

ax2.set_title('SecAuto Request/Response Data Flow', fontsize=18, fontweight='bold', pad=20)
ax2.axis('off')

# Save the flow diagram
flow_output_file = '/Volumes/My Shared Files/Home/Downloads/SecAuto/secauto_dataflow_graph.png'
plt.savefig(flow_output_file, dpi=300, bbox_inches='tight', facecolor='white')
print(f"Data flow graph saved to: {flow_output_file}")

plt.show()