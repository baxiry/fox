package tchecker

import (
	"fox/aster"
	"strings"
)

func (tc *TypeChecker) checkMatchStmt(s *aster.MatchStmt) {
	if s == nil || s.Object == nil {
		return
	}

	objType := tc.inferType(s.Object)
	if objType == nil {
		return
	}

	isErrorEnvelope := strings.HasPrefix(objType.Name, "_Result_")

	for _, c := range s.Cases {
		isErrorCase := false
		if len(c.Conditions) > 0 {
			if ident, ok := c.Conditions[0].(*aster.IdentExpr); ok && ident.Name == "Error" {
				isErrorCase = true
			}
		}

		if isErrorEnvelope {
			if isErrorCase {
				originalType := objType.Name
				objType.Name = "Error"

				if c.Body != nil {
					tc.checkBlock(c.Body)
				}

				objType.Name = originalType
				continue
			} else {
				originalType := objType.Name
				objType.Name = strings.TrimPrefix(originalType, "_Result_")

				if c.Body != nil {
					tc.checkBlock(c.Body)
				}

				objType.Name = originalType
				continue
			}
		}

		if c.Body != nil {
			tc.checkBlock(c.Body)
		}
	}

	if s.Else != nil {
		if isErrorEnvelope {
			originalType := objType.Name
			objType.Name = strings.TrimPrefix(originalType, "_Result_")

			tc.checkBlock(s.Else)

			objType.Name = originalType
		} else {
			tc.checkBlock(s.Else)
		}
	}
}

// end
