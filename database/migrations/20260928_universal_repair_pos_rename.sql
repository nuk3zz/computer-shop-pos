-- Rename only the former product placeholder. Real shop names remain unchanged.
UPDATE shop_profile
SET company_name = 'Universal Repair POS',
    description = CASE
        WHEN description = 'Sales and repair management' THEN 'Sales, service, and repair management'
        ELSE description
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE company_name = 'Computer Shop POS';
