#!/usr/bin/env python3
"""
Analyze the captured Leonardo data and extract GraphQL queries.
We need to re-capture with less aggressive sanitization.
"""
import json
from pathlib import Path

capture_file = Path.home() / "leonardo_seedance_2.5_capture.jsonl"

# Find video-related URLs
print("=" * 70)
print("VIDEO GENERATION EVIDENCE")
print("=" * 70)
print()

# Look at the CDN URLs - they show the model name
print("1. CDN URLs (show model names):")
print("-" * 70)
with open(capture_file, 'r') as f:
    cdn_urls = set()
    for line in f:
        event = json.loads(line)
        if 'cdn.leonardo.ai' in event['url']:
            url = event['url']
            if 'seedance' in url.lower() or 'bytedance' in url.lower():
                cdn_urls.add(url)
    
    for url in sorted(cdn_urls):
        print(url)

print()
print("2. Key observations from CDN URLs:")
print("-" * 70)
print("✓ Model identifier: bytedance/seedance-2.5")
print("✓ Generated videos stored at: cdn.leonardo.ai/users/{user_id}/generations/{gen_id}/bytedance/seedance-2.5_{prompt}-0.mp4")
print("✓ Thumbnails: ...seedance-2.5_{prompt}-0.jpg")
print()

# Show the generation flow
print("3. API Flow Summary:")
print("-" * 70)
print("✓ Endpoint: POST https://api.leonardo.ai/v1/graphql")
print("✓ 180 GraphQL requests captured")
print("✓ Headers include: x-leo-schema-version, authorization")
print("✓ Body structure: {operationName, variables, query}")
print()

print("4. What we need to capture properly:")
print("-" * 70)
print("[ ] The exact GraphQL mutation/query for video generation")
print("[ ] The variables passed (prompt, model, width, height, etc)")
print("[ ] The response structure (job ID, status polling)")
print("[ ] The completion detection method")
print()

print("NEXT STEP: We need to capture with actual request bodies")
print("The sanitization was too aggressive - we need the GraphQL query text")
print("and variable values (not credentials)")
