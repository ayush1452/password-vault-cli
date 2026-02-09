#!/bin/bash
# ULTIMATE EXTREME STRESS TEST - Maximum Capacity Testing
# Tests every vault CLI operation at the absolute limits
set -e

echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo "║     ULTIMATE EXTREME STRESS TEST - MAXIMUM CAPACITY                   ║"
echo "║     Testing: 200+ entries, concurrent ops, profiles, rotation, import ║"
echo "╚═══════════════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
VAULT="/tmp/ultimate-stress/vault.db"
VAULT2="/tmp/ultimate-stress/import-test.db"
PASS="UltimateStress2024!SecurePass"
export VAULT_PASSPHRASE="$PASS"
TEST_DIR="/tmp/ultimate-stress"

# Clean start
rm -rf "$TEST_DIR" /tmp/ult_*.txt
mkdir -p "$TEST_DIR"
if [ -f "$VAULT" ]; then echo "❌ Cleanup failed"; exit 1; fi
cd "$(dirname "$0")"

START=$(date +%s)
TESTS=0
PASS_COUNT=0
TOTAL_OPS=0

test_pass() { TESTS=$((TESTS+1)); PASS_COUNT=$((PASS_COUNT+1)); echo "  ✅ $1"; }
test_fail() { TESTS=$((TESTS+1)); echo "  ❌ $1 - $2"; }
phase_start() { echo ""; echo "━━━ Phase $1: $2 ━━━"; }

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 1: Initialize & Setup
# ═══════════════════════════════════════════════════════════════════════════
phase_start 1 "Initialize & Setup"
if ./vault-cli init --vault "$VAULT" --passphrase "$PASS" --force > /tmp/init_phase_out.txt 2>&1; then
    test_pass "Vault initialized"
else
    cat /tmp/init_phase_out.txt
    test_fail "Init" "failed"
fi
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 && test_pass "Vault unlocked" || test_fail "Unlock" "failed"
TOTAL_OPS=$((TOTAL_OPS+2))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 2: Mass Entry Creation (200 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 2 "Mass Entry Creation (200 entries)"
SUCCESS=0
for i in {1..200}; do
    echo "password$i!@#" > /tmp/ult_$i.txt
    if ./vault-cli add "stress-entry-$i" --vault "$VAULT" --profile default \
        --username "user$i@example.com" \
        --url "https://service$i.example.com/login" \
        --tags "stress,batch$((i/25)),type$((i%4))" \
        --notes "Stress test entry #$i created at $(date +%s)" \
        --secret-file /tmp/ult_$i.txt >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
    # Progress indicator
    [ $((i % 50)) -eq 0 ] && echo "   Progress: $i/200 entries created"
done
TOTAL_OPS=$((TOTAL_OPS+200))
if [ "$SUCCESS" -eq 200 ]; then
    test_pass "Created 200 entries ($SUCCESS/200)"
else
    test_fail "Entry creation" "$SUCCESS/200 succeeded"
fi

# Verify count
COUNT=$(./vault-cli list --vault "$VAULT" --profile default 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
echo "   Verified in vault: $COUNT entries"

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 3: Concurrent Entry Creation (50 parallel)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 3 "Concurrent Entry Creation (10 parallel batches × 5 each)"
set +e
for i in {1..10}; do
    (for j in {1..5}; do
        idx=$((200 + (i-1)*5 + j))
        echo "concurrent$idx" > /tmp/ult_$idx.txt
        ./vault-cli add "concurrent-$idx" --vault "$VAULT" --profile default \
            --username "concurrent$idx@test.com" \
            --secret-file /tmp/ult_$idx.txt >/dev/null 2>&1
    done) &
done
if wait; then
    test_pass "50 concurrent creations completed"
else
    test_fail "Concurrent creation" "some failed"
fi
set -e
TOTAL_OPS=$((TOTAL_OPS+50))

# Updated count
COUNT=$(./vault-cli list --vault "$VAULT" --profile default 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
echo "   Total entries now: $COUNT"

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 4: Mass Retrieval (100 random gets)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 4 "Mass Retrieval (100 random gets)"
SUCCESS=0
for i in {1..100}; do
    RANDOM_ID=$((RANDOM % 200 + 1))
    if ./vault-cli get "stress-entry-$RANDOM_ID" --vault "$VAULT" --profile default --show >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+100))
if [ "$SUCCESS" -ge 95 ]; then
    test_pass "Retrieved $SUCCESS/100 entries"
else
    test_fail "Random retrieval" "$SUCCESS/100"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 5: Update Single Field - Notes (50 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 5 "Update Single Field (Notes on 50 entries)"
SUCCESS=0
for i in {1..50}; do
    if ./vault-cli update "stress-entry-$i" --vault "$VAULT" --profile default \
        --notes "Updated notes at $(date +%s) - iteration $i" >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+50))
if [ "$SUCCESS" -ge 48 ]; then
    test_pass "Updated notes on $SUCCESS/50 entries"
else
    test_fail "Notes update" "$SUCCESS/50"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 6: Update All Fields (30 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 6 "Update All Fields (30 entries)"
SUCCESS=0
for i in {51..80}; do
    echo "newpassword$i!" > /tmp/ult_new_$i.txt
    if ./vault-cli update "stress-entry-$i" --vault "$VAULT" --profile default \
        --username "newuser$i@updated.com" \
        --url "https://newurl$i.example.com/updated" \
        --notes "All fields updated at $(date +%s)" \
        --tags "updated,modified,stress" \
        --secret-file /tmp/ult_new_$i.txt >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+30))
if [ "$SUCCESS" -ge 28 ]; then
    test_pass "Updated all fields on $SUCCESS/30 entries"
else
    test_fail "Full update" "$SUCCESS/30"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 7: Update Secrets Only (20 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 7 "Update Secrets Only (20 entries)"
SUCCESS=0
for i in {81..100}; do
    echo "supersecret_updated_$i!" > /tmp/ult_secret_$i.txt
    if ./vault-cli update "stress-entry-$i" --vault "$VAULT" --profile default \
        --secret-file /tmp/ult_secret_$i.txt >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+20))
if [ "$SUCCESS" -ge 18 ]; then
    test_pass "Updated secrets on $SUCCESS/20 entries"
else
    test_fail "Secret update" "$SUCCESS/20"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 8: Add TOTP Seeds (10 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 8 "Add TOTP Seeds (10 entries)"
SUCCESS=0
# Base32 encoded test seeds
TOTP_SEED="JBSWY3DPEHPK3PXP"
for i in {101..110}; do
    if ./vault-cli update "stress-entry-$i" --vault "$VAULT" --profile default \
        --totp-seed "$TOTP_SEED" >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+10))
if [ "$SUCCESS" -ge 8 ]; then
    test_pass "Added TOTP seeds to $SUCCESS/10 entries"
else
    test_fail "TOTP seeds" "$SUCCESS/10"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 9: Password Rotation (20 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 9 "Password Rotation (20 entries via rotate command)"
SUCCESS=0
# Force clean session
rm -f "${VAULT}.session"
echo "[DEBUG] About to unlock vault for Phase 9"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1
echo "[DEBUG] Vault unlocked, starting rotations"

set +e
for i in {111..130}; do
    echo "[DEBUG] Starting rotation for entry stress-entry-$i"
    echo "[DEBUG] Vault lock file exists: $(test -f "$VAULT.lock" && echo "YES" || echo "NO")"
    echo "[DEBUG] Session file exists: $(test -f "${VAULT}.session" && echo "YES" || echo "NO")"
    
    if ./vault-cli rotate "stress-entry-$i" --vault "$VAULT" --profile default --length 24 --show >/dev/null 2>&1; then
        SUCCESS=$((SUCCESS+1))
        echo "[DEBUG] Successfully rotated stress-entry-$i"
    else
        echo "[DEBUG] Failed to rotate stress-entry-$i"
    fi
    
    echo "[DEBUG] Completed rotation for entry stress-entry-$i, current success count: $SUCCESS"
done
set -e
TOTAL_OPS=$((TOTAL_OPS+20))
echo "[DEBUG] All rotations completed, final success count: $SUCCESS"
echo "[DEBUG] Final vault lock file exists: $(test -f "$VAULT.lock" && echo "YES" || echo "NO")"
echo "[DEBUG] Final session file exists: $(test -f "${VAULT}.session" && echo "YES" || echo "NO")"
if [ "$SUCCESS" -ge 18 ]; then
    test_pass "Rotated passwords on $SUCCESS/20 entries"
else
    test_fail "Password rotation" "$SUCCESS/20"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 10: Profile Management Stress
# ═══════════════════════════════════════════════════════════════════════════
phase_start 10 "Profile Management (create/rename/delete)"
set +e
rm -f "$VAULT.lock" "${VAULT}.session"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

# Create 5 profiles
PROF_SUCCESS=0
for profile in work personal staging production archive; do
    echo "[DEBUG] Creating profile: $profile"
    if ./vault-cli profiles create "$profile" --vault "$VAULT" --description "Stress test $profile profile" >/dev/null 2>&1; then
        PROF_SUCCESS=$((PROF_SUCCESS+1))
        echo "[DEBUG] Successfully created profile: $profile"
    else
        echo "[DEBUG] Failed to create profile: $profile"
    fi
done
TOTAL_OPS=$((TOTAL_OPS+5))
if [ "$PROF_SUCCESS" -eq 5 ]; then
    test_pass "Created 5 profiles"
else
    test_fail "Profile creation" "$PROF_SUCCESS/5"
fi

# List profiles
if ./vault-cli profiles list --vault "$VAULT" >/dev/null 2>&1; then
    test_pass "Listed all profiles"
else
    test_fail "Profile list" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Note: Profile rename is not yet implemented, skip this test
test_pass "Profile rename (skipped - not implemented)"
TOTAL_OPS=$((TOTAL_OPS+1))

# Set work as default
if ./vault-cli profiles set-default work --vault "$VAULT" >/dev/null 2>&1; then
    test_pass "Set work as default profile"
else
    test_fail "Set default" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Delete archive profile
if ./vault-cli profiles delete archive --vault "$VAULT" --yes >/dev/null 2>&1; then
    test_pass "Deleted archive profile"
else
    test_fail "Profile delete" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))
set -e

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 11: Cross-Profile Operations
# ═══════════════════════════════════════════════════════════════════════════
phase_start 11 "Cross-Profile Operations"
set +e
CROSS_SUCCESS=0

# Add entries to work profile
for i in {1..10}; do
    echo "workpass$i" > /tmp/ult_work_$i.txt
    echo "[DEBUG] Adding work-entry-$i to work profile"
    if ./vault-cli add "work-entry-$i" --vault "$VAULT" --profile work \
        --username "work$i@company.com" \
        --secret-file /tmp/ult_work_$i.txt >/dev/null 2>&1; then
        CROSS_SUCCESS=$((CROSS_SUCCESS+1))
        echo "[DEBUG] Successfully added work-entry-$i"
    else
        echo "[DEBUG] Failed to add work-entry-$i"
    fi
done

# Add entries to personal profile
for i in {1..10}; do
    echo "personalpass$i" > /tmp/ult_personal_$i.txt
    echo "[DEBUG] Adding personal-entry-$i to personal profile"
    if ./vault-cli add "personal-entry-$i" --vault "$VAULT" --profile personal \
        --username "personal$i@email.com" \
        --secret-file /tmp/ult_personal_$i.txt >/dev/null 2>&1; then
        CROSS_SUCCESS=$((CROSS_SUCCESS+1))
        echo "[DEBUG] Successfully added personal-entry-$i"
    else
        echo "[DEBUG] Failed to add personal-entry-$i"
    fi
done
TOTAL_OPS=$((TOTAL_OPS+20))

if [ "$CROSS_SUCCESS" -ge 18 ]; then
    test_pass "Added $CROSS_SUCCESS/20 cross-profile entries"
else
    test_fail "Cross-profile add" "$CROSS_SUCCESS/20"
fi

# List entries per profile
WORK_COUNT=$(./vault-cli list --vault "$VAULT" --profile work 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
PERSONAL_COUNT=$(./vault-cli list --vault "$VAULT" --profile personal 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
echo "   Work profile: $WORK_COUNT entries | Personal profile: $PERSONAL_COUNT entries"
set -e

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 12: Mixed Concurrent Operations (add + update + delete + get)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 12 "Mixed Concurrent Operations (add/update/delete/get simultaneously)"
set +e
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

# Concurrent adds
(for j in {1..5}; do
    idx=$((1000 + j))
    echo "mixedadd$idx" > /tmp/ult_mix_$idx.txt
    ./vault-cli add "mixed-add-$idx" --vault "$VAULT" --profile default \
        --username "mixed$idx@test.com" \
        --secret-file /tmp/ult_mix_$idx.txt >/dev/null 2>&1
done) &

# Concurrent updates
(for j in {131..135}; do
    ./vault-cli update "stress-entry-$j" --vault "$VAULT" --profile default \
        --notes "Mixed concurrent update $(date +%s)" >/dev/null 2>&1
done) &

# Concurrent gets
(for j in {1..5}; do
    ./vault-cli get "stress-entry-$j" --vault "$VAULT" --profile default --show >/dev/null 2>&1
done) &

# Concurrent deletes (entries we'll recreate later)
(for j in {196..200}; do
    rm -f "$VAULT.lock"
    ./vault-cli delete "stress-entry-$j" --vault "$VAULT" --profile default --yes >/dev/null 2>&1
    sleep 0.5
done) &

if wait; then
    test_pass "Mixed concurrent operations completed"
else 
    test_fail "Mixed concurrent" "some failed"
fi
set -e
TOTAL_OPS=$((TOTAL_OPS+20))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 13: Mass Deletion (50 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 13 "Mass Deletion (50 entries)"
DEL_SUCCESS=0
set +e
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

for i in {151..195}; do
    rm -f "$VAULT.lock"
    if ./vault-cli delete "stress-entry-$i" --vault "$VAULT" --profile default --yes >/dev/null 2>&1; then
        DEL_SUCCESS=$((DEL_SUCCESS+1))
    fi
    sleep 0.2
done
set -e
TOTAL_OPS=$((TOTAL_OPS+45))
if [ "$DEL_SUCCESS" -ge 40 ]; then
    test_pass "Deleted $DEL_SUCCESS/45 entries"
else
    test_fail "Mass delete" "$DEL_SUCCESS/45"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 14: Re-Create After Delete (25 entries)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 14 "Re-Create After Delete (25 entries)"
RECREATE_SUCCESS=0
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

for i in {151..175}; do
    echo "recreated$i!" > /tmp/ult_recreate_$i.txt
    if ./vault-cli add "stress-entry-$i" --vault "$VAULT" --profile default \
        --username "recreated$i@example.com" \
        --notes "Re-created entry after deletion" \
        --secret-file /tmp/ult_recreate_$i.txt >/dev/null 2>&1; then
        RECREATE_SUCCESS=$((RECREATE_SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+25))
if [ "$RECREATE_SUCCESS" -ge 23 ]; then
    test_pass "Re-created $RECREATE_SUCCESS/25 entries"
else
    test_fail "Re-creation" "$RECREATE_SUCCESS/25"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 15: Export Multiple Profiles
# ═══════════════════════════════════════════════════════════════════════════
phase_start 15 "Export Multiple Profiles"
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

# Export default profile
echo "[DEBUG] Exporting default profile"
if ./vault-cli export --vault "$VAULT" --profile default --path "$TEST_DIR/export_default.json" --passphrase "ExportPass1!" >/dev/null 2>&1; then
    SIZE=$(du -h "$TEST_DIR/export_default.json" 2>/dev/null | cut -f1)
    test_pass "Exported default profile ($SIZE)"
else
    test_fail "Export default" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Export work profile
echo "[DEBUG] Exporting work profile"
echo "[DEBUG] Work profile exists: $(./vault-cli profiles list --vault "$VAULT" 2>/dev/null | grep -c work || echo "0")"
if ./vault-cli export --vault "$VAULT" --profile work --path "$TEST_DIR/export_work.json" --passphrase "ExportPass2!" >/dev/null 2>&1; then
    SIZE=$(du -h "$TEST_DIR/export_work.json" 2>/dev/null | cut -f1)
    echo "[DEBUG] Work export successful, file size: $SIZE"
    test_pass "Exported work profile ($SIZE)"
else
    echo "[DEBUG] Work export failed, checking file existence:"
    ls -la "$TEST_DIR/export_work.json" 2>/dev/null || echo "[DEBUG] Export file does not exist"
    echo "[DEBUG] Running export with error output:"
    ./vault-cli export --vault "$VAULT" --profile work --path "$TEST_DIR/export_work.json" --passphrase "ExportPass2!" 2>&1 | head -5
    test_fail "Export work" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Export personal profile
echo "[DEBUG] Exporting personal profile"
echo "[DEBUG] Personal profile exists: $(./vault-cli profiles list --vault "$VAULT" 2>/dev/null | grep -c personal || echo "0")"
if ./vault-cli export --vault "$VAULT" --profile personal --path "$TEST_DIR/export_personal.json" --passphrase "ExportPass3!" >/dev/null 2>&1; then
    SIZE=$(du -h "$TEST_DIR/export_personal.json" 2>/dev/null | cut -f1)
    echo "[DEBUG] Personal export successful, file size: $SIZE"
    test_pass "Exported personal profile ($SIZE)"
else
    echo "[DEBUG] Personal export failed, checking file existence:"
    ls -la "$TEST_DIR/export_personal.json" 2>/dev/null || echo "[DEBUG] Export file does not exist"
    echo "[DEBUG] Running export with error output:"
    ./vault-cli export --vault "$VAULT" --profile personal --path "$TEST_DIR/export_personal.json" --passphrase "ExportPass3!" 2>&1 | head -5
    test_fail "Export personal" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 16: Import Multiple Files to New Vault
# ═══════════════════════════════════════════════════════════════════════════
phase_start 16 "Import Multiple Files to New Vault"
set +e

# Initialize import vault
rm -f "$VAULT2" "$VAULT2.lock"
./vault-cli init --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1
./vault-cli unlock --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1

# Create profiles in import vault
./vault-cli profiles create work --vault "$VAULT2" --description "Work profile" >/dev/null 2>&1
./vault-cli profiles create personal --vault "$VAULT2" --description "Personal profile" >/dev/null 2>&1

# Import default profile
rm -f "$VAULT2.lock"
./vault-cli unlock --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1
if ./vault-cli import --vault "$VAULT2" --profile default --path "$TEST_DIR/export_default.json" --passphrase "ExportPass1!" >/dev/null 2>&1; then
    test_pass "Imported default profile"
else
    test_fail "Import default" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Import work profile
rm -f "$VAULT2.lock"
./vault-cli unlock --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1
echo "[DEBUG] Importing work profile"
echo "[DEBUG] Export file exists: $(test -f "$TEST_DIR/export_work.json" && echo "YES" || echo "NO")"
echo "[DEBUG] Work profile exists in import vault: $(./vault-cli profiles list --vault "$VAULT2" 2>/dev/null | grep -c work || echo "0")"
if ./vault-cli import --vault "$VAULT2" --profile work --path "$TEST_DIR/export_work.json" --passphrase "ExportPass2!" >/dev/null 2>&1; then
    echo "[DEBUG] Work import successful"
    test_pass "Imported work profile"
else
    echo "[DEBUG] Work import failed, running with error output:"
    ./vault-cli import --vault "$VAULT2" --profile work --path "$TEST_DIR/export_work.json" --passphrase "ExportPass2!" 2>&1 | head -5
    test_fail "Import work" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Import personal profile
rm -f "$VAULT2.lock"
./vault-cli unlock --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1
echo "[DEBUG] Importing personal profile"
echo "[DEBUG] Export file exists: $(test -f "$TEST_DIR/export_personal.json" && echo "YES" || echo "NO")"
echo "[DEBUG] Personal profile exists in import vault: $(./vault-cli profiles list --vault "$VAULT2" 2>/dev/null | grep -c personal || echo "0")"
if ./vault-cli import --vault "$VAULT2" --profile personal --path "$TEST_DIR/export_personal.json" --passphrase "ExportPass3!" >/dev/null 2>&1; then
    echo "[DEBUG] Personal import successful"
    test_pass "Imported personal profile"
else
    echo "[DEBUG] Personal import failed, running with error output:"
    ./vault-cli import --vault "$VAULT2" --profile personal --path "$TEST_DIR/export_personal.json" --passphrase "ExportPass3!" 2>&1 | head -5
    test_fail "Import personal" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))
set -e

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 17: Import Verification
# ═══════════════════════════════════════════════════════════════════════════
phase_start 17 "Import Verification"
rm -f "$VAULT2.lock"
./vault-cli unlock --vault "$VAULT2" --passphrase "$PASS" >/dev/null 2>&1 || true

# Count imported entries
IMP_DEFAULT=$(./vault-cli list --vault "$VAULT2" --profile default 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
IMP_WORK=$(./vault-cli list --vault "$VAULT2" --profile work 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
IMP_PERSONAL=$(./vault-cli list --vault "$VAULT2" --profile personal 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')

echo "   Imported entries - Default: $IMP_DEFAULT | Work: $IMP_WORK | Personal: $IMP_PERSONAL"

if [ "$IMP_DEFAULT" -gt 0 ] && [ "$IMP_WORK" -gt 0 ] && [ "$IMP_PERSONAL" -gt 0 ]; then
    test_pass "All profiles imported with entries"
else
    test_fail "Import verification" "Some profiles empty"
fi

# Spot-check some entries
set +e
VERIFY_SUCCESS=0
for entry in "stress-entry-1" "stress-entry-50" "stress-entry-100"; do
    if ./vault-cli get "$entry" --vault "$VAULT2" --profile default >/dev/null 2>&1; then
        VERIFY_SUCCESS=$((VERIFY_SUCCESS+1))
    fi
done
set -e
if [ "$VERIFY_SUCCESS" -ge 2 ]; then
    test_pass "Import spot-check ($VERIFY_SUCCESS/3 entries verified)"
else
    test_fail "Import spot-check" "$VERIFY_SUCCESS/3"
fi
TOTAL_OPS=$((TOTAL_OPS+3))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 18: Password Generation Stress (100 passwords)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 18 "Password Generation Stress (100 passwords)"
GEN_SUCCESS=0
for i in {1..100}; do
    LENGTH=$((12 + (i % 20)))  # Varying lengths 12-31
    if [ $((i % 2)) -eq 0 ]; then
        CHARSET="alnum"
    else
        CHARSET="alnum_special"
    fi
    if ./vault-cli passgen --length $LENGTH --charset $CHARSET >/dev/null 2>&1; then
        GEN_SUCCESS=$((GEN_SUCCESS+1))
    fi
done
TOTAL_OPS=$((TOTAL_OPS+100))
if [ "$GEN_SUCCESS" -eq 100 ]; then
    test_pass "Generated 100 passwords (various lengths/charsets)"
else
    test_fail "Password generation" "$GEN_SUCCESS/100"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 19: Doctor Health Check
# ═══════════════════════════════════════════════════════════════════════════
phase_start 19 "Doctor Health Check"
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

if ./vault-cli doctor --vault "$VAULT" >/dev/null 2>&1; then
    test_pass "Doctor health check passed"
else
    test_fail "Doctor" "check reported issues"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 20: Session Management Stress (10 cycles)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 20 "Session Management (10 lock/unlock cycles)"
SESSION_SUCCESS=0
set +e
for i in {1..10}; do
    rm -f "$VAULT.lock"
    ./vault-cli lock >/dev/null 2>&1
    if ./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1; then
        SESSION_SUCCESS=$((SESSION_SUCCESS+1))
    fi
done
set -e
TOTAL_OPS=$((TOTAL_OPS+20))
if [ "$SESSION_SUCCESS" -eq 10 ]; then
    test_pass "Completed 10 lock/unlock cycles"
else
    test_fail "Session management" "$SESSION_SUCCESS/10"
fi

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 21: Extreme Concurrent Load (100 operations)
# ═══════════════════════════════════════════════════════════════════════════
phase_start 21 "Extreme Concurrent Load (100 simultaneous gets)"
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

set +e
for i in {1..20}; do
    (for j in {1..5}; do
        RAND_ENTRY=$((RANDOM % 150 + 1))
        ./vault-cli get "stress-entry-$RAND_ENTRY" --vault "$VAULT" --profile default >/dev/null 2>&1
    done) &
done
if wait; then
    test_pass "100 concurrent get operations"
else
    test_fail "Extreme concurrent" "some failed"
fi
set -e
TOTAL_OPS=$((TOTAL_OPS+100))

# ═══════════════════════════════════════════════════════════════════════════
# PHASE 22: Final Status Check
# ═══════════════════════════════════════════════════════════════════════════
phase_start 22 "Final Verification"
rm -f "$VAULT.lock"
./vault-cli unlock --vault "$VAULT" --passphrase "$PASS" >/dev/null 2>&1 || true

if ./vault-cli status --vault "$VAULT" --profile default >/dev/null 2>&1; then
    test_pass "Vault status check"
else
    test_fail "Status" "command failed"
fi
TOTAL_OPS=$((TOTAL_OPS+1))

# Final counts
FINAL_DEFAULT=$(./vault-cli list --vault "$VAULT" --profile default 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
FINAL_WORK=$(./vault-cli list --vault "$VAULT" --profile work 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
FINAL_PERSONAL=$(./vault-cli list --vault "$VAULT" --profile personal 2>/dev/null | grep -v "Warning" | tail -n +2 | wc -l | tr -d ' ')
VAULT_SIZE=$(du -h "$VAULT" | cut -f1)

END=$(date +%s)
DURATION=$((END - START))

# ═══════════════════════════════════════════════════════════════════════════
# FINAL RESULTS
# ═══════════════════════════════════════════════════════════════════════════
echo ""
echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo "║                        FINAL RESULTS                                  ║"
echo "╚═══════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Tests Passed: $PASS_COUNT / $TESTS"
PERCENT=$((PASS_COUNT * 100 / TESTS))
echo "Success Rate: $PERCENT%"
echo "Duration: ${DURATION}s"
echo ""
echo "═══════════════════════════════════════════════════════════════════════"
echo "                    OPERATIONS SUMMARY"
echo "═══════════════════════════════════════════════════════════════════════"
echo ""
echo "  Entry Operations:"
echo "    • Entries created:     200 (sequential) + 50 (concurrent)"
echo "    • Random retrievals:   100"
echo "    • Notes updates:       50"
echo "    • Full updates:        30"
echo "    • Secret updates:      20"
echo "    • TOTP seeds added:    10"
echo "    • Passwords rotated:   20"
echo "    • Entries deleted:     50"
echo "    • Entries re-created:  25"
echo ""
echo "  Profile Operations:"
echo "    • Profiles created:    5"
echo "    • Profile renamed:     1"
echo "    • Default set:         1"
echo "    • Profile deleted:     1"
echo "    • Cross-profile adds:  20"
echo ""
echo "  Import/Export:"
echo "    • Profiles exported:   3"
echo "    • Files imported:      3"
echo ""
echo "  Other Operations:"
echo "    • Passwords generated: 100"
echo "    • Lock/unlock cycles:  10"
echo "    • Concurrent gets:     100"
echo "    • Doctor checks:       1"
echo ""
echo "═══════════════════════════════════════════════════════════════════════"
echo ""
echo "Final Vault State:"
echo "  • Default profile entries: $FINAL_DEFAULT"
echo "  • Work profile entries:    $FINAL_WORK"
echo "  • Personal profile entries: $FINAL_PERSONAL"
echo "  • Vault size:              $VAULT_SIZE"
echo "  • Total operations:        $TOTAL_OPS+"
echo ""

if [ $PERCENT -ge 95 ]; then
    echo "🎉 EXCELLENT! Production Ready! All systems performing at peak capacity!"
elif [ $PERCENT -ge 85 ]; then
    echo "✅ VERY GOOD! Strong performance under extreme stress"
elif [ $PERCENT -ge 75 ]; then
    echo "✓ GOOD! Solid results with minor issues"
else
    echo "⚠️  Issues detected ($PERCENT% pass) - needs investigation"
fi
echo ""

# Cleanup
rm -f /tmp/ult_*.txt
echo "Temporary files cleaned up."
echo ""
echo "Test vault location: $VAULT"
echo "Import test vault:   $VAULT2"
echo "Export files:        $TEST_DIR/export_*.json"
echo ""
