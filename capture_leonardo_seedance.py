#!/usr/bin/env python3
"""
Capture Leonardo.ai Seedance 2.5 network requests.
User logs in and manually generates a video while we passively capture the API calls.
"""
import asyncio
import json
import re
from pathlib import Path
from datetime import datetime
from playwright.async_api import async_playwright

# Output file
CAPTURE_FILE = Path.home() / "leonardo_seedance_2.5_capture.jsonl"
SUMMARY_FILE = Path.home() / "leonardo_seedance_2.5_summary.txt"

# Profile for persistent login
PROFILE_DIR = Path.home() / "tracking" / "profiles" / "leonardo"
PROFILE_DIR.mkdir(parents=True, exist_ok=True)

captured_events = []

def sanitize_headers(headers):
    """Keep header names but redact sensitive values"""
    safe_headers = {}
    sensitive = ['authorization', 'cookie', 'set-cookie', 'x-api-key', 'x-auth-token']
    for k, v in headers.items():
        if k.lower() in sensitive:
            safe_headers[k] = "[REDACTED]"
        else:
            safe_headers[k] = v
    return safe_headers

def sanitize_url(url):
    """Remove tokens and signed parameters from URLs"""
    # Remove common token patterns
    url = re.sub(r'[?&](token|key|sig|signature|auth)=[^&]+', r'\1=[REDACTED]', url)
    return url

def sanitize_body(body_text, content_type):
    """Extract structure but redact sensitive values"""
    if not body_text:
        return None
    
    try:
        if 'json' in content_type.lower():
            data = json.loads(body_text)
            # Keep structure, note sensitive fields
            return {"_shape": describe_json_shape(data), "_sample_keys": list(data.keys())[:10] if isinstance(data, dict) else None}
        else:
            return {"_type": content_type, "_length": len(body_text)}
    except:
        return {"_type": content_type, "_length": len(body_text) if body_text else 0}

def describe_json_shape(obj, path="", depth=0, max_depth=4):
    """Recursively describe JSON structure without exposing values"""
    if depth > max_depth:
        return "..."
    
    if isinstance(obj, dict):
        shape = {}
        for k, v in obj.items():
            new_path = f"{path}.{k}" if path else k
            shape[k] = describe_json_shape(v, new_path, depth + 1, max_depth)
        return shape
    elif isinstance(obj, list):
        if len(obj) == 0:
            return []
        # Show first item structure
        return [describe_json_shape(obj[0], f"{path}[0]", depth + 1, max_depth)]
    else:
        return type(obj).__name__

async def main():
    print("=" * 70)
    print("Leonardo.ai Seedance 2.5 Network Capture")
    print("=" * 70)
    print()
    print("This script will:")
    print("1. Open Leonardo.ai in a visible browser")
    print("2. Capture all network requests while you work")
    print("3. Save sanitized API call data")
    print()
    print("INSTRUCTIONS:")
    print("- Login to Leonardo.ai if needed")
    print("- Navigate to video generation")
    print("- Select Seedance 2.5 model")
    print("- Generate a video")
    print("- Wait for it to complete")
    print("- When done, close the browser window")
    print()
    print(f"Profile directory: {PROFILE_DIR}")
    print(f"Capture output: {CAPTURE_FILE}")
    print(f"Summary output: {SUMMARY_FILE}")
    print()
    print("🚀 Launching browser in 3 seconds...")
    await asyncio.sleep(3)
    
    async with async_playwright() as p:
        # Launch persistent context (keeps cookies)
        context = await p.chromium.launch_persistent_context(
            user_data_dir=str(PROFILE_DIR),
            headless=False,
            viewport={'width': 1920, 'height': 1080},
            args=['--disable-blink-features=AutomationControlled']
        )
        
        page = context.pages[0] if context.pages else await context.new_page()
        
        # Request handler - capture all requests
        async def handle_request(request):
            event = {
                "timestamp": datetime.now().isoformat(),
                "type": "request",
                "method": request.method,
                "url": sanitize_url(request.url),
                "headers": sanitize_headers(request.headers),
                "resource_type": request.resource_type
            }
            
            # Try to get POST body
            if request.method in ["POST", "PUT", "PATCH"]:
                try:
                    post_data = request.post_data
                    content_type = request.headers.get('content-type', '')
                    event["body"] = sanitize_body(post_data, content_type)
                except:
                    pass
            
            captured_events.append(event)
            
            # Print interesting requests
            if any(keyword in request.url.lower() for keyword in ['video', 'generate', 'seedance', 'motion', 'api']):
                print(f"📤 {request.method} {request.url[:100]}")
        
        # Response handler - capture responses
        async def handle_response(response):
            request = response.request
            event = {
                "timestamp": datetime.now().isoformat(),
                "type": "response",
                "method": request.method,
                "url": sanitize_url(request.url),
                "status": response.status,
                "headers": sanitize_headers(response.headers),
            }
            
            # Try to get response body for API calls
            if any(keyword in request.url.lower() for keyword in ['api', 'graphql']) and response.status < 400:
                try:
                    content_type = response.headers.get('content-type', '')
                    if 'json' in content_type.lower():
                        body = await response.text()
                        event["body"] = sanitize_body(body, content_type)
                except:
                    pass
            
            captured_events.append(event)
            
            # Print interesting responses
            if any(keyword in request.url.lower() for keyword in ['video', 'generate', 'seedance', 'motion', 'api']):
                print(f"📥 {response.status} {request.method} {request.url[:100]}")
        
        page.on("request", handle_request)
        page.on("response", handle_response)
        
        # Navigate to Leonardo.ai
        print("\n🌐 Opening Leonardo.ai...")
        await page.goto("https://app.leonardo.ai/", wait_until="domcontentloaded")
        
        print("\n✅ READY - Network capture is active!")
        print("\nNow:")
        print("1. Login if needed")
        print("2. Go to video generation")
        print("3. Select Seedance 2.5")
        print("4. Generate a video and wait for completion")
        print("5. Close the browser when done")
        print()
        
        # Keep browser open until user closes it
        try:
            while context.pages:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            print("\n\n⚠️  Interrupted by user")
        
        print("\n💾 Saving captured data...")
        
        # Save all events as JSONL
        with open(CAPTURE_FILE, 'w') as f:
            for event in captured_events:
                f.write(json.dumps(event) + '\n')
        
        # Create human-readable summary
        with open(SUMMARY_FILE, 'w') as f:
            f.write("Leonardo.ai Seedance 2.5 Network Capture Summary\n")
            f.write("=" * 70 + "\n\n")
            f.write(f"Capture time: {datetime.now().isoformat()}\n")
            f.write(f"Total events: {len(captured_events)}\n\n")
            
            # Group by URL pattern
            api_calls = {}
            for event in captured_events:
                if event['type'] == 'request':
                    url = event['url']
                    # Extract base pattern
                    base = re.sub(r'/[0-9a-f-]{20,}', '/{id}', url)
                    base = re.sub(r'\?.*', '', base)
                    if base not in api_calls:
                        api_calls[base] = []
                    api_calls[base].append(event)
            
            f.write("API Endpoints Found:\n")
            f.write("-" * 70 + "\n")
            for url_pattern, calls in sorted(api_calls.items()):
                if any(k in url_pattern.lower() for k in ['api', 'graphql', 'video', 'generate', 'seedance']):
                    methods = set(c['method'] for c in calls)
                    f.write(f"\n{url_pattern}\n")
                    f.write(f"  Methods: {', '.join(methods)}\n")
                    f.write(f"  Calls: {len(calls)}\n")
                    
                    # Show a sample request
                    sample = calls[0]
                    if 'body' in sample and sample['body']:
                        f.write(f"  Body structure: {json.dumps(sample['body'], indent=4)}\n")
        
        print(f"\n✅ Capture saved!")
        print(f"   Events: {CAPTURE_FILE}")
        print(f"   Summary: {SUMMARY_FILE}")
        print(f"   Total captured: {len(captured_events)} events")
        print()
        print("Next steps:")
        print(f"1. Review the summary: cat {SUMMARY_FILE}")
        print(f"2. Look for video generation endpoints")
        print(f"3. Identify the Seedance 2.5 model parameter")
        print(f"4. I'll help you implement it in your local API")

if __name__ == "__main__":
    asyncio.run(main())
