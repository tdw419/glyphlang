#!/usr/bin/env python3
"""
GlyphLang Token Compression Analyzer

Measures token savings when using GlyphLang vs natural language for AIPM workflows.
"""

import re
from dataclasses import dataclass
from typing import List, Tuple


@dataclass
class CompressionResult:
    original: str
    compressed: str
    original_tokens: int
    compressed_tokens: int
    savings_percent: float
    compression_type: str


# GlyphLang symbol mappings (token-efficient)
SYMBOL_MAPPINGS = {
    # Natural language → GlyphLang symbol
    r'\bCreate a function\b': '!',
    r'\bDefine a function\b': '!',
    r'\bAdd a route\b': '@',
    r'\bDefine a type\b': ':',
    r'\bSet variable\b': '$',
    r'\bReturn\b': '>',
    r'\bPrint\b': 'print',
    r'\bIf\b': 'if',
    r'\bWhile\b': 'while',
    r'\bFor each\b': 'for',
    r'\bImport\b': 'import',
    r'\bExport\b': 'export',
}

# Workflow compression patterns
WORKFLOW_PATTERNS = [
    # AIPM task patterns
    (r'Enqueue a task with priority (\d+) that says "([^"]+)"', 
     r'enqueue("\2", \1)'),
    (r'Process the next task in the queue', 
     r'dequeue()'),
    (r'Get the current queue statistics', 
     r'queue_stats()'),
    (r'Mark task (\d+) as completed with result "([^"]+)"', 
     r'complete(\1, "\2")'),
    
    # Common operations
    (r'Call the LLM with prompt "([^"]+)" and return the result', 
     r'llm("\1")'),
    (r'Execute the GlyphLang code "([^"]+)"', 
     r'exec("\1")'),
    (r'Parse the response and extract the result field', 
     r'.result'),
    (r'Loop through all items and process each one', 
     r'for item in items { process(item) }'),
    
    # Spatial/GPU patterns
    (r'Encode the thought as Hilbert coordinates at position', 
     r'hilbert_encode(thought)'),
    (r'Render the pixels to the framebuffer', 
     r'render(pixels)'),
    (r'Spawn (\d+) parallel agents to process', 
     r'S \1 spawn'),
    (r'Self-modify the code to optimize', 
     r'M optimize'),
]


def count_tokens(text: str) -> int:
    """Rough token count (words + symbols)"""
    # Split on whitespace and punctuation
    tokens = re.findall(r'\b\w+\b|[{}()\[\]$@:!?<>]', text)
    return len(tokens)


def compress_workflow(natural_language: str) -> CompressionResult:
    """Compress natural language workflow to GlyphLang"""
    compressed = natural_language
    
    # Apply workflow patterns
    for pattern, replacement in WORKFLOW_PATTERNS:
        compressed = re.sub(pattern, replacement, compressed)
    
    # Apply symbol mappings
    for pattern, replacement in SYMBOL_MAPPINGS.items():
        compressed = re.sub(pattern, replacement, compressed)
    
    # Count tokens
    original_tokens = count_tokens(natural_language)
    compressed_tokens = count_tokens(compressed)
    
    if original_tokens > 0:
        savings = (1 - compressed_tokens / original_tokens) * 100
    else:
        savings = 0
    
    return CompressionResult(
        original=natural_language,
        compressed=compressed,
        original_tokens=original_tokens,
        compressed_tokens=compressed_tokens,
        savings_percent=savings,
        compression_type="workflow"
    )


def analyze_batch(examples: List[str]) -> List[CompressionResult]:
    """Analyze compression for multiple examples"""
    return [compress_workflow(ex) for ex in examples]


# Demo examples
DEMO_EXAMPLES = [
    # AIPM workflows
    'Enqueue a task with priority 8 that says "Implement P1 security fixes"',
    'Process the next task in the queue and call the LLM with the prompt',
    'Get the current queue statistics and print the pending count',
    
    # Complex orchestrations
    'Loop through all items and process each one with the LLM and save results',
    'Spawn 5 parallel agents to process the queue and aggregate results',
    'Self-modify the code to optimize for token efficiency',
    
    # Spatial workflows
    'Encode the thought as Hilbert coordinates at position and render pixels',
    
    # Multi-step workflows
    'Create a function that takes a string and reverses it, then return the result',
    'Add a route that handles GET requests to /users and returns the user list',
    'Define a type called User with name as string and age as integer fields',
]


def print_report(results: List[CompressionResult]):
    """Print compression analysis report"""
    print("=" * 70)
    print("GlyphLang Token Compression Analysis")
    print("=" * 70)
    print()
    
    total_original = 0
    total_compressed = 0
    
    for i, r in enumerate(results, 1):
        print(f"Example {i}:")
        print(f"  Original ({r.original_tokens} tokens):")
        print(f"    {r.original[:60]}..." if len(r.original) > 60 else f"    {r.original}")
        print(f"  Compressed ({r.compressed_tokens} tokens):")
        print(f"    {r.compressed[:60]}..." if len(r.compressed) > 60 else f"    {r.compressed}")
        print(f"  Savings: {r.savings_percent:.1f}%")
        print()
        
        total_original += r.original_tokens
        total_compressed += r.compressed_tokens
    
    # Summary
    if total_original > 0:
        overall_savings = (1 - total_compressed / total_original) * 100
    else:
        overall_savings = 0
    
    print("=" * 70)
    print("Summary")
    print("=" * 70)
    print(f"Total original tokens:   {total_original}")
    print(f"Total compressed tokens: {total_compressed}")
    print(f"Overall savings:         {overall_savings:.1f}%")
    print()
    
    # By compression level
    high_savings = [r for r in results if r.savings_percent >= 50]
    medium_savings = [r for r in results if 25 <= r.savings_percent < 50]
    low_savings = [r for r in results if r.savings_percent < 25]
    
    print(f"High savings (≥50%):    {len(high_savings)} examples")
    print(f"Medium savings (25-50%): {len(medium_savings)} examples")
    print(f"Low savings (<25%):      {len(low_savings)} examples")


if __name__ == "__main__":
    results = analyze_batch(DEMO_EXAMPLES)
    print_report(results)
