UPDATE "membership_plan"
SET "name" = '年费会员',
    "description" = '开通后可购买订阅商品，默认有效期为一年。',
    "updated_at" = NOW()
WHERE "id" = 1
  AND "name" = 'Annual Membership'
  AND "description" = 'Default annual membership card. Required before purchasing subscription products.';
