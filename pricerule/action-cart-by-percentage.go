package pricerule

import "log"

// CalculateDiscountsCartByPercentage -
func calculateDiscountsCartByPercentage(articleCollection *ArticleCollection, priceRuleVoucherPair RuleVoucherPair, orderDiscounts OrderDiscounts, productGroupIDsPerPosition map[string][]string, groupIDsForCustomer []string, roundTo float64, isCatalogCalculation bool) OrderDiscounts {
	if priceRuleVoucherPair.Rule.Action != ActionCartByPercent {
		panic("CalculateDiscountsCartByPercentage called with pricerule of action " + priceRuleVoucherPair.Rule.Action)
	}

	if isCatalogCalculation == true {
		log.Println("catalog calculations can not handle actions of type CalculateDiscountsCartByPercentage")
		return orderDiscounts
	}

	//get the total - for vouchers it is lowered by previous discounts
	orderTotal := getOrderTotalForPriceRule(priceRuleVoucherPair.Rule, articleCollection, productGroupIDsPerPosition, groupIDsForCustomer, orderDiscounts)
	//the discount amount calculation
	totalDiscountAmount := roundToStep(orderTotal*priceRuleVoucherPair.Rule.Amount/100.0, roundTo)

	//from here we call existing methods with a hacked priceRule that will keep the name and ID but different action and amount
	tempPriceRule := *priceRuleVoucherPair.Rule
	tempPriceRule.Action = ActionCartByAbsolute
	tempPriceRule.Amount = totalDiscountAmount
	tempPriceRuleVoucherPair := RuleVoucherPair{Rule: &tempPriceRule, Voucher: priceRuleVoucherPair.Voucher}

	return calculateDiscountsCartByAbsolute(articleCollection, tempPriceRuleVoucherPair, orderDiscounts, productGroupIDsPerPosition, groupIDsForCustomer, roundTo, isCatalogCalculation)
}
