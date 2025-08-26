"""
SecAuto SDK Data Models

Data classes and models for representing SecAuto API responses
and request structures.
"""

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any, Union
from datetime import datetime


@dataclass
class Job:
    """Represents a job in the SecAuto system."""
    id: str
    status: str
    playbook: Any
    context: Dict[str, Any]
    results: Optional[Any] = None
    error: Optional[str] = None
    created_at: Optional[datetime] = None
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    priority: int = 0
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class PlaybookRequest:
    """Request structure for playbook execution."""
    context: Dict[str, Any]
    playbook: Optional[Any] = None
    playbook_name: Optional[str] = None


@dataclass
class PlaybookResponse:
    """Response structure for playbook execution."""
    success: bool
    message: Optional[str] = None
    result: Optional[Any] = None
    job_id: Optional[str] = None
    timestamp: Optional[str] = None


@dataclass
class APIKey:
    """Represents an API key."""
    key: str
    name: str
    description: str
    created_at: str
    created_by: str
    active: bool
    source: str
    last_used: Optional[str] = None


@dataclass
class APIKeySummary:
    """Summary representation of an API key (without full key)."""
    key_prefix: str
    name: str
    description: str
    created_at: str
    created_by: str
    active: bool
    source: str
    last_used: Optional[str] = None


@dataclass
class Client:
    """Represents a client in the system."""
    id: str
    name: str
    description: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
    enabled: bool = True
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class Integration:
    """Represents an integration configuration."""
    name: str
    type: str
    enabled: bool
    config: Dict[str, Any]
    credentials: Optional[Dict[str, Any]] = None
    client_id: Optional[str] = None
    shared_with: List[str] = field(default_factory=list)
    access_level: Optional[str] = None
    encryption_key_id: Optional[str] = None
    inherit_from: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


@dataclass
class JobSchedule:
    """Represents a scheduled job."""
    id: str
    name: str
    description: str
    cron_expression: str
    playbook: Any
    context: Dict[str, Any]
    enabled: bool
    created_at: str
    updated_at: str
    next_run: Optional[str] = None
    last_run: Optional[str] = None
    status: str = "enabled"


@dataclass
class AutomationInfo:
    """Information about an automation script."""
    name: str
    filename: str
    size: int
    file_type: str
    language: str
    line_count: int
    function_count: int
    import_count: int
    modified_at: str
    is_valid: bool


@dataclass
class AutomationMetadata:
    """Metadata for an automation script."""
    name: str
    description: str
    version: str
    author: str
    category: str
    tags: List[str]
    parameters: List[Dict[str, Any]]
    dependencies: List[str]
    created_at: str
    updated_at: str
    config: Dict[str, Any] = field(default_factory=dict)


@dataclass
class CacheStats:
    """Cache statistics."""
    context_hits: int
    context_misses: int
    expression_hits: int
    expression_misses: int
    variable_hits: int
    variable_misses: int
    evicted_contexts: int
    evicted_expressions: int
    evicted_variables: int
    total_size: int
    cleanup_runs: int


@dataclass
class JobStats:
    """Job statistics."""
    total_jobs: int
    completed: int
    failed: int
    running: int
    pending: int
    avg_duration_seconds: float
    recent_jobs: List[Job] = field(default_factory=list)


@dataclass
class APIResponse:
    """Generic API response wrapper."""
    success: bool
    message: Optional[str] = None
    data: Optional[Any] = None
    timestamp: Optional[str] = None
    errors: List[Dict[str, Any]] = field(default_factory=list)
