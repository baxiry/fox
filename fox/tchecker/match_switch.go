package tchecker

import (
	"fox/aster"
)

func (tc *TypeChecker) checkMatchStmt(s *aster.MatchStmt) {
	if s == nil || s.Object == nil {
		return
	}

	// 1. استنباط والتحقق من نوع المتغير المستهدف بالفحص (مثل s)
	_ = tc.inferType(s.Object)

	// 2. حلقة معزولة تماماً لزيارة وفحص الأنواع والجمل داخل كل فرع case
	for _, c := range s.Cases {
		// فحص كتلة الجمل والتعليمات التابعة للفرع الحالي باستخدام دالتك القياسية
		if c.Body != nil {
			tc.checkBlock(c.Body)
		}
	}

	// 3. فحص الفرع البديل الافتراضي else إن وجد
	if s.Else != nil {
		tc.checkBlock(s.Else)
	}
}
