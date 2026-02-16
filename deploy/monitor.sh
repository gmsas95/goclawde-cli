#!/bin/bash
#
# Myrai Monitoring Script
# Checks health and sends alerts if needed
#

MYRAI_URL="${MYRAI_URL:-http://localhost:8080}"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"
LOG_FILE="${LOG_FILE:-/var/log/myrai/monitor.log}"

# Create log directory
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_health() {
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "$MYRAI_URL/api/health" 2>/dev/null || echo "000")
    
    if [ "$response" = "200" ]; then
        log "✅ Health check passed"
        return 0
    else
        log "❌ Health check failed (HTTP $response)"
        return 1
    fi
}

check_disk_space() {
    local usage
    usage=$(df /var/lib/myrai | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [ "$usage" -gt 90 ]; then
        log "⚠️ Disk usage critical: ${usage}%"
        return 1
    elif [ "$usage" -gt 80 ]; then
        log "⚠️ Disk usage warning: ${usage}%"
    else
        log "✅ Disk usage OK: ${usage}%"
    fi
    return 0
}

check_memory() {
    local usage
    usage=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100.0}')
    
    if [ "$usage" -gt 90 ]; then
        log "⚠️ Memory usage critical: ${usage}%"
        return 1
    elif [ "$usage" -gt 80 ]; then
        log "⚠️ Memory usage warning: ${usage}%"
    else
        log "✅ Memory usage OK: ${usage}%"
    fi
    return 0
}

send_alert() {
    local message="$1"
    
    if [ -n "$ALERT_WEBHOOK" ]; then
        curl -s -X POST -H "Content-Type: application/json" \
            -d "{\"text\":\"$message\"}" \
            "$ALERT_WEBHOOK" > /dev/null
    fi
}

# Main monitoring
log "🔍 Starting Myrai health check..."

failed=0

if ! check_health; then
    failed=1
    send_alert "🚨 Myrai health check failed on $(hostname)"
fi

if ! check_disk_space; then
    failed=1
fi

if ! check_memory; then
    failed=1
fi

if [ $failed -eq 0 ]; then
    log "✅ All checks passed"
    exit 0
else
    log "❌ Some checks failed"
    exit 1
fi
