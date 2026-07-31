UPDATE "membership_plan"
SET "name" = 'Annual Membership',
    "description" = 'Default annual membership card. Required before purchasing subscription products.',
    "updated_at" = NOW()
WHERE "id" = 1
  AND "name" = '年费会员'
  AND "description" = '开通后可购买订阅商品，默认有效期为一年。';
