#!/usr/bin/env python3
"""Test Python Temporal connection"""

import asyncio
import os
import sys
from temporalio.client import Client

async def test_connection():
    print("=== Testing Python Temporal Connection ===")
    
    # Test with localhost
    print("1. Testing with localhost:7233...")
    try:
        client = await Client.connect("localhost:7233")
        print("✓ localhost connection successful")
        # Python client doesn't need explicit close
    except Exception as e:
        print(f"✗ localhost connection failed: {e}")
    
    # Test with IPv4 address
    print("2. Testing with 127.0.0.1:7233...")
    try:
        client = await Client.connect("127.0.0.1:7233")
        print("✓ 127.0.0.1 connection successful")
        
        # Test basic functionality
        try:
            # Try to get service info
            service_info = await client.service_info()
            print(f"✓ Health check successful - Server version: {service_info.server_version}")
        except Exception as e:
            print(f"Health check failed: {e}")
        
    except Exception as e:
        print(f"✗ 127.0.0.1 connection failed: {e}")

if __name__ == "__main__":
    # Clear proxy environment variables
    for key in ['http_proxy', 'https_proxy', 'all_proxy', 
                'HTTP_PROXY', 'HTTPS_PROXY', 'ALL_PROXY']:
        if key in os.environ:
            del os.environ[key]
    
    asyncio.run(test_connection())