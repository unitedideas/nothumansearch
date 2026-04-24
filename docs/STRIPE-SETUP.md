# NHS /fix/{host} Live Payment Setup

**Status**: Code ready. Blocked on Stripe key + webhook config.
**Effort**: ~10 min. Follow checklist below.
**Owner**: Shane (Stripe Dashboard access required)

---

## Overview

NHS `/fix/{host}` endpoint is live in **lead-capture mode** (manual invoicing). To flip to live direct payment:

1. ✅ Code ready (handlers/fix.go + webhook handler registered at `/webhook/stripe`)
2. ⏳ Stripe config: Create restricted key + webhook (this doc)
3. ⏳ Fly secrets: Set `STRIPE_SECRET_KEY` + `STRIPE_WEBHOOK_SECRET`
4. ⏳ E2E smoke test: Test /fix flow end-to-end

**Current behavior** (STRIPE_SECRET_KEY unset):
- User submits `/fix/{host}` form → recorded as `status='lead'` → Discord alert → manual Stripe invoice sent
- No direct payment, no checkout redirect

**After setup**:
- User submits `/fix/{host}` form → Stripe Checkout session created → user redirected to Stripe → payment completes → webhook marks `status='paid'` + Discord alert

---

## Step 1: Create Restricted Stripe Key

**Destination**: 8Bit Stripe account (`acct_1TNfT83svHq2QCOI`)

1. Go to [Stripe Dashboard](https://dashboard.stripe.com/account/api-keys) (logged in as 8Bit account)
2. Click **"Create restricted key"** → **"Create restricted key"** again
3. Name: `NHS live payment (direct checkout)`
4. Permissions (click "Write" or "Read" for each scope):
   | Scope | Permission | Reason |
   |---|---|---|
   | Products | Write | Product creation for checkout |
   | Prices | Write | Price creation for checkout |
   | Customers | Write | Store customer email on Stripe |
   | Checkout Sessions | Write | Create checkout sessions |
   | Payment Links | Write | Future payment link support |
   | Subscriptions | Write | Future recurring billing |
   | Billing Portal | Write | Customer portal (future) |
   | Webhook Endpoints | Write | Register webhook endpoint |
   | Charges | Read | Webhook events require read |
   | Payment Intents | Read | Webhook events require read |
   | Events | Read | Webhook signature verification |
   | All other scopes | None | (Leave as None) |

5. **Copy the key** → it starts with `rk_live_...` → paste into Step 3 below

---

## Step 2: Create Webhook Endpoint

**Destination**: 8Bit Stripe account, Webhook Endpoints

1. Go to [Stripe Webhooks](https://dashboard.stripe.com/webhooks)
2. Click **"Add an endpoint"**
3. Endpoint URL: `https://nothumansearch.ai/webhook/stripe`
4. Events to send:
   - ✅ `checkout.session.completed` (REQUIRED — marks job as paid)
   - ✅ `checkout.session.expired` (payment canceled/expired)
   - ✅ `payment_intent.payment_failed` (payment failed)
   - ✅ `charge.refunded` (refund issued)
   - ✅ `customer.subscription.created` (future recurring)
   - ✅ `customer.subscription.updated` (future recurring)
   - ✅ `customer.subscription.deleted` (future recurring)
   - ✅ `invoice.payment_failed` (future recurring)

5. Click **"Add endpoint"**
6. After creation, click the endpoint → copy the **"Signing secret"** (starts with `whsec_`)

---

## Step 3: Set Fly Secrets

Run this command **once** with the keys from Steps 1 & 2:

```bash
fly secrets set \
  STRIPE_SECRET_KEY=rk_live_... \
  STRIPE_WEBHOOK_SECRET=whsec_... \
  -a nothumansearch
```

**Verification** (Fly will auto-restart the nothumansearch app):
```bash
fly status -a nothumansearch
```

Wait 10–30s for the app to redeploy + health checks to pass (green status).

---

## Step 4: E2E Smoke Test

**Prerequisites**: Complete Steps 1–3 above. Fly app should be healthy.

```bash
# Start a test run
TEST_HOST="example.com"
echo "Testing /fix/$TEST_HOST endpoint..."

# 1. Fetch the intake form (should load, no 500 errors)
curl -s "https://nothumansearch.ai/fix/$TEST_HOST" | grep -q "Pay \$199" && echo "✅ Form loads" || echo "❌ Form missing"

# 2. POST to create Stripe session (should redirect to Stripe)
SESSION=$(curl -s -X POST "https://nothumansearch.ai/fix/$TEST_HOST" \
  -d "email=test@example.com&repo_url=https://github.com/example/site&notes=test" \
  -L -w "%{redirect_url}" 2>/dev/null)

echo "Session redirect: $SESSION"
if echo "$SESSION" | grep -q "checkout.stripe.com"; then
  echo "✅ Stripe redirect works"
else
  echo "❌ No Stripe redirect (check Fly logs: fly logs -a nothumansearch)"
fi

# 3. Check Fly logs for errors
echo ""
echo "Recent Fly logs (check for 'fix: CreateGeoFixJob' success):"
fly logs -a nothumansearch --limit 50 2>/dev/null | grep -E "(fix:|Stripe|checkout)" || echo "(No recent fix logs yet)"
```

**What to look for**:
- ✅ Form loads at `/fix/{host}` (not 500, not Stripe key error)
- ✅ Form POST redirects to `checkout.stripe.com` (Stripe-hosted checkout)
- ✅ Fly logs show `fix: CreateGeoFixJob` + `gostripe.New()` calls
- ✅ No "STRIPE_SECRET_KEY not set" warnings in logs

**Test a real payment** (optional, uses test card):
1. Form POST redirects to Stripe Checkout
2. Use Stripe test card: `4242 4242 4242 4242` / any future date / any CVC
3. Complete payment → should redirect to `/fix/success?id=...&session_id=...`
4. Check Discord: should show `💰 **NHS fix-my-score paid** — ...`
5. Check DB: `SELECT * FROM geo_fix_jobs WHERE status='paid' ORDER BY id DESC LIMIT 1;`

---

## Rollback

If something breaks:

```bash
# Remove Stripe keys (revert to lead-capture mode)
fly secrets unset STRIPE_SECRET_KEY STRIPE_WEBHOOK_SECRET -a nothumansearch

# Or set to empty (explicit unset)
fly secrets set STRIPE_SECRET_KEY= STRIPE_WEBHOOK_SECRET= -a nothumansearch
```

App will auto-restart and fall back to `status='lead'` + manual invoicing.

---

## Reference

- **Code**: `/internal/handlers/fix.go` — intake form, Stripe session creation, webhook handler
- **Database schema**: `migrations/004_geo_fix_jobs.sql` — stores jobs with status: `pending`, `lead`, `paid`
- **Current flow** (active until secrets are set): POST → CreateGeoFixJob → status='lead' → Discord ping → manual invoice
- **Post-setup flow**: POST → CreateGeoFixJob + Stripe session → Checkout → webhook → MarkGeoFixJobPaid → status='paid' → Discord ping

---

## Timing

**Owner**: Shane (10 min via Stripe Dashboard)
**Pre-requisites**: None (all code ready)
**Impact**: NHS revenue stream unlocked ($199 × N orders)
**Risk**: Low (live Stripe account already active, isolated to NHS /fix endpoint only)
