# lago-fork-dhh: Invoice Totals, Finalization Numbering, and Parity Suite

## Objective
Implement fee aggregation → coupon/credit deduction → tax computation → subtotals → total math, sequential numbering on finalization, and GraphQL resolvers for Invoice operations. Parity with Rails `ComputeAmountsFromFees`.

## Calculation Formula

```
fees_amount_cents         = SUM(fee.amount_cents)
sub_total_excluding_taxes = fees - progressive_billing_credit - coupons
taxes_amount_cents        = CEIL(sub_total_ex * rate / 100)  [0 if sub_total ≤ 0]
sub_total_including_taxes = sub_total_ex + taxes
total_amount_cents        = sub_total_inc - credit_notes     [floor at 0]
```

## Sequential Numbering
- `organization_sequential_id` = `MAX(org invoices, status=finalized) + 1` in DB transaction
- `number` = `LAGO-{YYYYMM}-{seq:03d}`
- `issuing_date` = today UTC
- `payment_due_date` = issuing_date + net_payment_term days

## Files Created

| File | Purpose |
|------|---------|
| `api-go/migrations/000007_fees.up/down.sql` | Fees table DDL |
| `api-go/internal/models/fee.go` | Fee GORM model — all Rails columns, FeeType + FeePaymentStatus enums |
| `api-go/internal/domain/invoices/totals.go` | `CalculateTotals()` pure function |
| `api-go/internal/domain/invoices/totals_test.go` | 10 parity tests |

## Files Modified

| File | Change |
|------|--------|
| `services/invoices/invoice_service.go` | `Finalize` computes totals + assigns sequential ID + formats number in DB tx |
| `services/invoices/invoice_service_test.go` | Updated for new Finalize queries |
| `graphql/resolver.go` | Added `InvoiceSvc` |
| `graphql/schema.resolvers.go` | Implemented FinalizeInvoice, VoidInvoice, Invoice query, UpdateInvoice + helpers |
| `server/server.go` | Wired InvoiceSvc into resolver |

## Status
✅ **COMPLETE** — Commit `b20fe08`, pushed to `origin/main`. All 30 test packages pass.
Epic `lago-fork-59c` auto-closed.
