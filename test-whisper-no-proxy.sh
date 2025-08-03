#!/bin/bash
# Test whisper-server without proxy settings

# Clear all proxy settings
unset ALL_PROXY
unset all_proxy  
unset HTTP_PROXY
unset HTTPS_PROXY
unset http_proxy
unset https_proxy
unset NO_PROXY
unset no_proxy
unset SOCKS_PROXY
unset socks_proxy

# Run the test
./test-whisper-server-client -file test/data/test.mp3 -url "http://192.168.31.151:8080" -verbose