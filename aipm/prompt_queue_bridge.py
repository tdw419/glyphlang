#!/usr/bin/env python3
"""
AIPM-GlyphLang Bridge

Connects GlyphLang orchestrator to the AIPM SQLite database.
Enables GlyphLang to manage prompt queues with token-efficient syntax.
"""

import sqlite3
import json
import os
from pathlib import Path
from datetime import datetime

# AIPM database path
AIPM_DB = Path.home() / "zion" / "projects" / "ai_project_management" / "aipm" / "data" / "truths.db"

class AIPMBridge:
    def __init__(self, db_path=None):
        self.db_path = db_path or AIPM_DB
        self.conn = None
        
    def connect(self):
        """Connect to AIPM database"""
        if self.db_path.exists():
            self.conn = sqlite3.connect(str(self.db_path))
            return True
        return False
    
    def get_pending_prompts(self, limit=10):
        """Get pending prompts from queue"""
        if not self.conn:
            return []
        
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT id, prompt, priority, created_at
            FROM prompt_queue
            WHERE status = 'pending'
            ORDER BY priority DESC, created_at ASC
            LIMIT ?
        """, (limit,))
        
        return [
            {
                "id": row[0],
                "text": row[1],
                "priority": row[2],
                "created_at": row[3]
            }
            for row in cursor.fetchall()
        ]
    
    def enqueue_prompt(self, text, priority=5, source="glyphlang"):
        """Add prompt to queue"""
        if not self.conn:
            return None
        
        cursor = self.conn.cursor()
        cursor.execute("""
            INSERT INTO prompt_queue (prompt, priority, status, source, created_at)
            VALUES (?, ?, 'pending', ?, ?)
        """, (text, priority, source, datetime.now().isoformat()))
        self.conn.commit()
        
        return cursor.lastrowid
    
    def mark_processing(self, prompt_id):
        """Mark prompt as processing"""
        if not self.conn:
            return False
        
        cursor = self.conn.cursor()
        cursor.execute("""
            UPDATE prompt_queue
            SET status = 'processing', started_at = ?
            WHERE id = ?
        """, (datetime.now().isoformat(), prompt_id))
        self.conn.commit()
        
        return True
    
    def mark_completed(self, prompt_id, result):
        """Mark prompt as completed with result"""
        if not self.conn:
            return False
        
        cursor = self.conn.cursor()
        cursor.execute("""
            UPDATE prompt_queue
            SET status = 'completed', result = ?, completed_at = ?
            WHERE id = ?
        """, (json.dumps(result), datetime.now().isoformat(), prompt_id))
        self.conn.commit()
        
        return True
    
    def get_stats(self):
        """Get queue statistics"""
        if not self.conn:
            return {}
        
        cursor = self.conn.cursor()
        
        stats = {}
        for status in ['pending', 'processing', 'completed', 'failed']:
            cursor.execute("""
                SELECT COUNT(*) FROM prompt_queue WHERE status = ?
            """, (status,))
            stats[status] = cursor.fetchone()[0]
        
        return stats
    
    def close(self):
        if self.conn:
            self.conn.close()


def compress_to_glyphlang(natural_language: str) -> str:
    """
    Compress natural language prompt to GlyphLang syntax.
    Returns token savings percentage.
    """
    # Example compression patterns
    compressions = {
        "Create a function that": "! ",
        "Add a route that handles": "@ ",
        "Define a type called": ": ",
        "Return the result": "> ",
        "variable equals": "$ ",
        "print output": "print(",
        "if condition": "if ",
        "while loop": "while ",
        "and then": "\n  ",
        "finally": "\n  > ",
    }
    
    compressed = natural_language
    for pattern, replacement in compressions.items():
        compressed = compressed.replace(pattern, replacement)
    
    # Calculate savings
    original_tokens = len(natural_language.split())
    compressed_tokens = len(compressed.split())
    
    if original_tokens > 0:
        savings = (1 - compressed_tokens / original_tokens) * 100
    else:
        savings = 0
    
    return compressed, savings


# GlyphLang builtins for bridge
BUILTINS = {
    "aipm_enqueue": lambda bridge, text, priority: bridge.enqueue_prompt(text, priority),
    "aipm_dequeue": lambda bridge, limit: bridge.get_pending_prompts(limit),
    "aipm_stats": lambda bridge: bridge.get_stats(),
    "aipm_complete": lambda bridge, id, result: bridge.mark_completed(id, result),
}


if __name__ == "__main__":
    # Demo
    bridge = AIPMBridge()
    
    # Create in-memory DB for demo
    bridge.conn = sqlite3.connect(":memory:")
    bridge.conn.execute("""
        CREATE TABLE IF NOT EXISTS prompt_queue (
            id INTEGER PRIMARY KEY,
            prompt TEXT,
            priority INTEGER DEFAULT 5,
            status TEXT DEFAULT 'pending',
            source TEXT,
            result TEXT,
            created_at TEXT,
            started_at TEXT,
            completed_at TEXT
        )
    """)
    
    # Enqueue some tasks
    print("=== AIPM-GlyphLang Bridge Demo ===\n")
    
    id1 = bridge.enqueue_prompt("Implement P1 security fixes", 8)
    id2 = bridge.enqueue_prompt("Add bytecode execution", 7)
    id3 = bridge.enqueue_prompt("Update documentation", 5)
    
    print(f"Enqueued {id1}, {id2}, {id3}")
    
    # Get stats
    stats = bridge.get_stats()
    print(f"\nQueue Stats: {json.dumps(stats, indent=2)}")
    
    # Get pending
    pending = bridge.get_pending_prompts(5)
    print(f"\nPending ({len(pending)}):")
    for p in pending:
        print(f"  [{p['priority']}] {p['text']}")
    
    # Test compression
    print("\n=== Token Compression Demo ===")
    nl = "Create a function that takes a string and returns the reversed string"
    compressed, savings = compress_to_glyphlang(nl)
    print(f"Original: {nl}")
    print(f"Compressed: {compressed}")
    print(f"Savings: {savings:.1f}%")
    
    bridge.close()
