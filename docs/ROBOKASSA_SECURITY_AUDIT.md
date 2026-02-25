# RoboKassa Internal Security Audit

## Threat model
- Fake webhook calls to `/webhooks/robokassa/result` with forged signatures.
- Replay attacks by resubmitting valid callbacks.
- Amount tampering by changing `OutSum` in callback.
- Invoice collision attempts on `robokassa_invoice_id`.

## Mitigations implemented
- Signature validation for ResultURL uses RoboKassa Password #2 and rejects mismatches.
- Amount is validated against server-side persisted payment amount before status changes.
- Idempotent webhook processing via row-level lock and status checks in transaction.
- Unique constraint on `payments.robokassa_invoice_id` prevents invoice collision.
- Consistent DB transactions are used for status transitions and subscription activation.

## Compliance notes
- PCI-DSS scope reduced: no PAN/card/CVV data stored or processed.
- Sensitive merchant secrets are read from environment variables only.
- Test/prod password selection is dynamic based on `ROBOKASSA_IS_TEST`.
