package database

import (
	"log-backend/internal/domain"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// devPasswords holds the LOCAL TESTING credentials for the seeded demo
// accounts. They exist only so a fresh install (or an existing dev DB)
// can be exercised end-to-end with email/password login. Never reuse
// these in production — rotate them before any real deployment.
var devPasswords = map[string]string{
	"admin-1":  "Admin@123",
	"mod-1":    "Teacher@123",
	"user-123": "Student@123",
}

// strPtr returns a pointer to the given string, useful for optional unique fields.
func strPtr(s string) *string { return &s }

func InitDB() {
	var err error

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/log.db"
	}

	// Ensure the parent directory exists (skipped for absolute temp paths in tests)
	if !filepath.IsAbs(dbPath) {
		if dir := filepath.Dir(dbPath); dir != "." {
			if err := os.MkdirAll(dir, 0750); err != nil {
				slog.Error("Failed to create data directory:", "error", err)
				os.Exit(1)
			}
		}
	}

	// SQLite seam hardening: WAL + NORMAL for concurrent reads during writes
	// and a 5s busy timeout so concurrent teacher/student writes never surface
	// SQLITE_BUSY. Note: the _foreign_keys=on pragma is honored by SQLite, but
	// gorm.AutoMigrate does NOT create FK constraints for the models below
	// (they declare no association tags), so referential integrity is enforced
	// at the service layer. Do not document this file as "FK-enforced" until
	// the models carry real constraints.
	//
	// WP-0.1 research: _secure_delete=ON (mattn driver) makes SQLite overwrite
	// deleted/updated cell content with zeros inside b-tree pages. It does NOT
	// scrub the WAL file — the erasure path (privacy_repo.ScrubDeletedData)
	// checkpoints and VACUUMs after account deletion to clear both. Backups
	// made before a VACUUM still contain deleted data; rotate them on erasure
	// (see docs/PRIVACY_RUNBOOK.md).
	DB, err = gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=on&_secure_delete=ON"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("Failed to connect to database:", "error", err)
		os.Exit(1)
	}

	// WP-4.2 pool sizing for SQLite: this engine is a single-writer. Pinning
	// the pool to ONE connection makes concurrent goroutines serialize on
	// SQLite's own locking instead of tripping SQLITE_BUSY, and removes
	// connection-split hazards entirely. Reads are WAL-fast and the app is
	// low-traffic, so the serialization cost is negligible on the low-end
	// school hardware LOG targets.
	sqlDB, err := DB.DB()
	if err != nil {
		slog.Error("Failed to reach sql pool:", "error", err)
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0) // SQLite connections are cheap file handles — never recycle mid-flight

	// Auto Migrate
	err = DB.AutoMigrate(
		&domain.User{},
		&domain.OTPRecord{},
		&domain.Activity{},
		&domain.LearnerActivity{},
		&domain.Progress{},
		&domain.Observation{},
		&domain.Guidance{},
		&domain.Course{},
		&domain.Enrollment{},
		&domain.DailyActivity{},
		&domain.MicroModule{},
		&domain.TokenBlocklist{},
		&domain.Class{},
		&domain.ClassMember{},
		&domain.Announcement{},
		&domain.Assignment{},
		&domain.Submission{},
		&domain.AuditLog{},
		&domain.UserRevocation{},
		&domain.ConsentRecord{},
		// Phase 2 (WP-2.1/2.2/2.3): parent links, support issues, notes.
		&domain.ParentLink{},
		&domain.SupportIssue{},
		&domain.LearnerNote{},
		// Phase 3 (WP-3.3 RC-10): QR poster pilot scans.
		&domain.PilotScan{},
	)
	if err != nil {
		slog.Error("Failed to migrate database:", "error", err)
		os.Exit(1)
	}

	// Purge expired blocklist entries on startup to keep the table lean
	DB.Where("expires_at < ?", time.Now()).Delete(&domain.TokenBlocklist{})

	seedData()

	// WP-4.1 C4: real, enforced FOREIGN KEY constraints. AutoMigrate (above)
	// cannot emit them, so MigrateForeignKeys rebuilds each affected table —
	// idempotent, and it skips tables with orphan rows rather than deleting
	// data. Runs after seeding so a stock DB is guaranteed orphan-free.
	MigrateForeignKeys(DB)
}

func seedData() {
	var count int64
	DB.Model(&domain.User{}).Count(&count)
	if count == 0 {
		// Seed Users
		users := []domain.User{
			{ID: "admin-1", Name: "Principal Skinner", Email: "admin@log.edu", Phone: strPtr("1000000000"), Role: domain.RoleAdmin, IsVerified: true},
			{ID: "mod-1", Name: "Teacher Edna", Email: "teacher@log.edu", Phone: strPtr("2000000000"), Role: domain.RoleModerator, IsVerified: true},
			{ID: "user-123", Name: "Aisha Student", Email: "aisha@example.com", Phone: strPtr("+9779800000000"), Role: domain.RoleStudent, IsVerified: true},
		}
		for _, u := range users {
			DB.Create(&u)
		}

		// Seed Progress & Acts
		DB.Create(&domain.Progress{LearnerID: "user-123", TotalTopics: 10, Completed: 2, CurrentStreak: 3, OverallScore: 85.5})
		acts := []domain.Activity{
			{ID: "act-1", Title: "Introduction to Logic", Description: "Basic concepts.", Topic: "Logic", Order: 1},
			{ID: "act-2", Title: "Boolean Algebra", Description: "AND, OR, NOT.", Topic: "Logic", Order: 2},
			// WP-1.2 RC-02: SEE 9-12 pilot units — curriculum-aligned practice
			// with a supportive explanation on every question.
			{ID: "act-3", Title: "SEE Mathematics: Quadratic Equations", Description: "Solve, factor, and graph quadratics step by step.", Topic: "Mathematics", Order: 3},
			{ID: "act-4", Title: "SEE Science: Electricity", Description: "Current, voltage, Ohm's law, and circuits.", Topic: "Science", Order: 4},
			{ID: "act-5", Title: "SEE English: Tenses", Description: "Master the 12 tenses with supportive practice.", Topic: "English", Order: 5},
		}
		for _, a := range acts {
			DB.Create(&a)
		}

		learnerActs := []domain.LearnerActivity{
			{LearnerID: "user-123", ActivityID: "act-1", Status: "Completed", CompletedAt: time.Now(), Score: 100},
			{LearnerID: "user-123", ActivityID: "act-2", Status: "In progress", Score: 50},
		}
		for _, la := range learnerActs {
			DB.Create(&la)
		}

		obs := []domain.Observation{
			{ID: "obs-1", LearnerID: "user-123", Category: "strengths", Text: "Strong grasp of Boolean Algebra."},
			{ID: "obs-2", LearnerID: "user-123", Category: "consistency", Text: "Studying consistently for 3 days."},
		}
		for _, o := range obs {
			DB.Create(&o)
		}

		gui := []domain.Guidance{
			{ID: "gui-1", LearnerID: "user-123", Type: "next_step", Text: "Continue Boolean Algebra.", Action: "/learning/act-2"},
		}
		for _, g := range gui {
			DB.Create(&g)
		}
	}

	// Ensure the seeded demo accounts can log in via email/password.
	// Runs on every startup but only writes when the hash is empty, so
	// existing dev databases get backfilled and a real user's hash is
	// never overwritten.
	for id, pw := range devPasswords {
		var u domain.User
		if err := DB.Where("id = ?", id).First(&u).Error; err != nil || u.PasswordHash != "" {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
		if err != nil {
			slog.Error("Failed to hash seed password", "user", id, "error", err)
			continue
		}
		DB.Model(&domain.User{}).Where("id = ?", id).Update("password_hash", string(hash))
	}

	// Seed Micro-Modules independently of users, so existing databases
	// also receive the bite-sized module content on next startup.
	// 30 modules across the two seeded activities: concept lessons plus
	// knowledge checks (real quiz data for the attempt seam).
	var mmCount int64
	DB.Model(&domain.MicroModule{}).Count(&mmCount)
	if mmCount == 0 {
		microModules := []domain.MicroModule{
			// ─── act-1: Introduction to Logic (14 modules) ───
			{ID: "mm-1", ActivityID: "act-1", Title: "What is Logic?", ContentText: "Logic is the study of reasoning. In computing, logic gates make every decision your computer makes — from checking a password to rendering a page.", Order: 1,
				Question: "Which of these is a proposition (a statement with a truth value)?", Options: []string{"Close the door!", "What time is it?", "Kathmandu is in Nepal", "Run faster!"}, CorrectIndex: 2,
				Explanation: "A proposition is a declarative sentence that is either true or false. \"Kathmandu is in Nepal\" is true, so it is a proposition."},
			{ID: "mm-2", ActivityID: "act-1", Title: "Truth Values", ContentText: "Every logical statement resolves to one of two values: True or False. Think of them as the two possible answers to any yes/no question.", Order: 2,
				Question: "How many truth values exist in classical logic?", Options: []string{"One", "Two", "Three", "Four"}, CorrectIndex: 1,
				Explanation: "Classical logic works with exactly two values: True and False."},
			{ID: "mm-3", ActivityID: "act-1", Title: "Propositions in Everyday Life", ContentText: "Sentences like \"It is raining\" and \"The bus is late\" are propositions. Questions and commands are not — they have no truth value.", Order: 3,
				Question: "Which sentence is NOT a proposition?", Options: []string{"The sky is blue", "2 + 2 = 4", "Please open the window", "The sun rises in the east"}, CorrectIndex: 2,
				Explanation: "\"Please open the window\" is a request, not a statement — it cannot be true or false."},
			{ID: "mm-4", ActivityID: "act-1", Title: "Negation (NOT)", ContentText: "Negation flips a truth value. NOT True is False, and NOT False is True. In algebra we write NOT A as ¬A or A'.", Order: 4,
				Question: "What is NOT (False)?", Options: []string{"False", "True", "Undefined", "Depends on the context"}, CorrectIndex: 1,
				Explanation: "Negation always flips the value, so NOT False is True."},
			{ID: "mm-5", ActivityID: "act-1", Title: "Conjunction (AND)", ContentText: "AND returns True only when BOTH inputs are True. A strict bouncer: you need an ID AND a ticket to enter.", Order: 5,
				Question: "If A is True and B is False, what is (A AND B)?", Options: []string{"True", "False", "Maybe", "Both"}, CorrectIndex: 1,
				Explanation: "AND is True only when both inputs are True. Since B is False, the whole statement is False."},
			{ID: "mm-6", ActivityID: "act-1", Title: "Disjunction (OR)", ContentText: "OR returns True when AT LEAST ONE input is True. A flexible cashier: you can pay with Cash OR Card.", Order: 6,
				Question: "What is (False OR True)?", Options: []string{"False", "True", "Neither", "Only if asked nicely"}, CorrectIndex: 1,
				Explanation: "OR needs only one True input. Since the second input is True, the result is True."},
			{ID: "mm-7", ActivityID: "act-1", Title: "Conditional (If... Then)", ContentText: "\"If it rains, the ground is wet\" is a conditional. It is only False when the premise is True and the conclusion is False (rain with a dry ground).", Order: 7,
				Question: "When is \"If P, then Q\" False?", Options: []string{"When P is False", "When Q is True", "When P is True and Q is False", "Never"}, CorrectIndex: 2,
				Explanation: "A conditional is broken only when the premise holds but the conclusion does not."},
			{ID: "mm-8", ActivityID: "act-1", Title: "Biconditional (If and Only If)", ContentText: "\"P if and only if Q\" means P and Q always agree: both True or both False. It is the logical \"equality\" operator.", Order: 8,
				Question: "Which pair makes (P iff Q) True?", Options: []string{"P=True, Q=False", "P=False, Q=True", "P=True, Q=True", "Any pair"}, CorrectIndex: 2,
				Explanation: "A biconditional is True exactly when both sides share the same truth value."},
			{ID: "mm-9", ActivityID: "act-1", Title: "Exclusive OR (XOR)", ContentText: "XOR is True when the inputs DIFFER: one is True, the other False. \"Would you like tea or coffee?\" — exactly one, not both.", Order: 9,
				Question: "What is (True XOR True)?", Options: []string{"True", "False", "True or False", "Undefined"}, CorrectIndex: 1,
				Explanation: "XOR is True only when the inputs differ. Two True inputs give False."},
			{ID: "mm-10", ActivityID: "act-1", Title: "Building Truth Tables", ContentText: "A truth table lists every possible input combination and its output. Two inputs give 4 rows; three inputs give 8; n inputs give 2ⁿ rows.", Order: 10,
				Question: "How many rows does a truth table with 3 inputs have?", Options: []string{"3", "6", "8", "9"}, CorrectIndex: 2,
				Explanation: "Each input doubles the combinations, so 2³ = 8 rows."},
			{ID: "mm-11", ActivityID: "act-1", Title: "Logical Equivalence", ContentText: "Two statements are logically equivalent when they have identical truth tables. \"A AND B\" is equivalent to \"B AND A\" — order does not matter.", Order: 11,
				Question: "Which is logically equivalent to (A AND B)?", Options: []string{"B AND A", "A OR B", "A AND (B OR C)", "NOT (A AND B)"}, CorrectIndex: 0,
				Explanation: "AND is commutative: swapping the inputs never changes the output."},
			{ID: "mm-12", ActivityID: "act-1", Title: "Quantifiers: All and Some", ContentText: "\"All students passed\" and \"Some students passed\" are quantified statements. All is stronger — it must hold for every single case.", Order: 12,
				Question: "If \"All students passed\" is True, which must also be True?", Options: []string{"Some students failed", "Some students passed", "No students passed", "The test was hard"}, CorrectIndex: 1,
				Explanation: "If everyone passed, then certainly at least one student passed."},
			{ID: "mm-13", ActivityID: "act-1", Title: "Common Reasoning Traps", ContentText: "Beware of fallacies: assuming the conclusion (\"It's true because it's true\") or assuming the opposite of a true statement is false (\"If P is true, NOT P must be false\" — that one is actually valid!).", Order: 13,
				Question: "Which is a valid logical move?", Options: []string{"Assuming what you want to prove", "If P is true, then NOT P is false", "Changing the topic mid-argument", "Trusting the loudest voice"}, CorrectIndex: 1,
				Explanation: "Negation genuinely flips truth values — that is a valid inference, not a fallacy."},
			{ID: "mm-14", ActivityID: "act-1", Title: "Logic in Computing", ContentText: "Every search filter, password check, and if-statement in your code is logic. Boolean thinking lets you combine conditions precisely instead of guessing.", Order: 14,
				Question: "Which real-world task is a logical AND?",
				Options:  []string{"\"Login if password is correct\"", "\"Login if password is correct AND account is active\"", "\"Login if username exists\"", "\"Display login button\""}, CorrectIndex: 1,
				Explanation: "AND requires both conditions — the password must be right AND the account must be active."},

			// ─── act-2: Boolean Algebra (16 modules) ───
			{ID: "mm-15", ActivityID: "act-2", Title: "The AND Operator", ContentText: "AND returns True only when BOTH inputs are True. In Boolean algebra we write A AND B as A·B or just AB.", Order: 1,
				Question: "What does (1 AND 0) equal in Boolean algebra?", Options: []string{"1", "0", "2", "Undefined"}, CorrectIndex: 1,
				Explanation: "AND needs both inputs to be 1. Since one input is 0, the result is 0."},
			{ID: "mm-16", ActivityID: "act-2", Title: "The OR Operator", ContentText: "OR returns True when AT LEAST ONE input is True, written A + B. It is True for three of the four input combinations.", Order: 2,
				Question: "What does (0 OR 1) equal?", Options: []string{"0", "1", "No value", "It depends"}, CorrectIndex: 1,
				Explanation: "OR is True when either input is 1 — here the second input is 1."},
			{ID: "mm-17", ActivityID: "act-2", Title: "The NOT Operator", ContentText: "NOT flips a value: 0 becomes 1 and 1 becomes 0. It is written as ¬A or A' (A-bar). NOT is the only single-input operator.", Order: 3,
				Question: "What is NOT(1)?", Options: []string{"1", "0", "10", "Undefined"}, CorrectIndex: 1,
				Explanation: "NOT flips every value, so NOT 1 is 0."},
			{ID: "mm-18", ActivityID: "act-2", Title: "The NAND Gate", ContentText: "NAND is AND followed by NOT: it is False only when BOTH inputs are True. It is the universal gate — any circuit can be built from NANDs.", Order: 4,
				Question: "What does NAND(1, 1) equal?", Options: []string{"1", "0", "2", "Undefined"}, CorrectIndex: 1,
				Explanation: "NAND(1,1) = NOT(AND(1,1)) = NOT(1) = 0."},
			{ID: "mm-19", ActivityID: "act-2", Title: "The NOR Gate", ContentText: "NOR is OR followed by NOT: it is True only when BOTH inputs are False. NOR is also a universal gate.", Order: 5,
				Question: "What does NOR(0, 0) equal?", Options: []string{"0", "1", "Undefined", "No value"}, CorrectIndex: 1,
				Explanation: "NOR(0,0) = NOT(OR(0,0)) = NOT(0) = 1."},
			{ID: "mm-20", ActivityID: "act-2", Title: "The XOR Gate", ContentText: "XOR is True when the inputs differ, written A ⊕ B. It powers binary addition: 1 + 1 = 0 with a carry of 1.", Order: 6,
				Question: "What does XOR(1, 0) equal?", Options: []string{"0", "1", "Carry only", "Undefined"}, CorrectIndex: 1,
				Explanation: "XOR is True when inputs differ — 1 and 0 differ, so the result is 1."},
			{ID: "mm-21", ActivityID: "act-2", Title: "The XNOR Gate", ContentText: "XNOR is the equality operator: True when both inputs are the same. It is the complement of XOR, written A ⊙ B.", Order: 7,
				Question: "What does XNOR(1, 1) equal?", Options: []string{"0", "1", "Depends on the gate", "Carry bit"}, CorrectIndex: 1,
				Explanation: "XNOR is True when the inputs match — both are 1, so the result is 1."},
			{ID: "mm-22", ActivityID: "act-2", Title: "Identity & Annulment Laws", ContentText: "A AND 1 = A (identity), A OR 0 = A (identity), A AND 0 = 0 (annulment), A OR 1 = 1 (annulment). These four laws are the arithmetic table of Boolean algebra.", Order: 8,
				Question: "What does (A AND 0) simplify to?", Options: []string{"A", "0", "1", "NOT A"}, CorrectIndex: 1,
				Explanation: "AND with 0 always gives 0 — the annulment law."},
			{ID: "mm-23", ActivityID: "act-2", Title: "Idempotent & Complement Laws", ContentText: "A AND A = A and A OR A = A (idempotent). A AND ¬A = 0 and A OR ¬A = 1 (complement). These simplify expressions by removing repeats.", Order: 9,
				Question: "What does (A OR ¬A) simplify to?", Options: []string{"0", "A", "1", "¬A"}, CorrectIndex: 2,
				Explanation: "A OR NOT A is always 1 — one of the two must be True (complement law)."},
			{ID: "mm-24", ActivityID: "act-2", Title: "Commutative & Associative Laws", ContentText: "Order and grouping do not matter: A·B = B·A, (A·B)·C = A·(B·C), and the same for OR. These laws let you rearrange expressions freely.", Order: 10,
				Question: "Which expression equals A·B?", Options: []string{"B·A", "A+B", "A·(B+C)", "¬A·B"}, CorrectIndex: 0,
				Explanation: "AND is commutative — swapping operands never changes the result."},
			{ID: "mm-25", ActivityID: "act-2", Title: "The Distributive Law", ContentText: "A·(B+C) = A·B + A·C, like ordinary algebra. Factoring a common term is the key trick in most simplifications.", Order: 11,
				Question: "Simplify A·B + A·C using the distributive law.", Options: []string{"A·(B+C)", "A·B·C", "A+B+C", "B·C"}, CorrectIndex: 0,
				Explanation: "A is common to both terms, so A·B + A·C = A·(B + C)."},
			{ID: "mm-26", ActivityID: "act-2", Title: "The Absorption Law", ContentText: "A + A·B = A and A·(A+B) = A. The extra term is absorbed away entirely — this law is easy to miss and very useful.", Order: 12,
				Question: "What does A + (A·B) simplify to?", Options: []string{"A·B", "A", "B", "A+B"}, CorrectIndex: 1,
				Explanation: "Absorption: A + A·B = A. The B term adds no new information."},
			{ID: "mm-27", ActivityID: "act-2", Title: "De Morgan's Theorem", ContentText: "NOT(A AND B) = (NOT A) OR (NOT B), and NOT(A OR B) = (NOT A) AND (NOT B). \"You can't be both\" becomes \"you lack one or the other.\"", Order: 13,
				Question: "What is ¬(A AND B) equivalent to?", Options: []string{"¬A AND ¬B", "¬A OR ¬B", "A OR B", "¬(A OR B)"}, CorrectIndex: 1,
				Explanation: "De Morgan flips the operator: NOT(AND) becomes OR with both inputs negated."},
			{ID: "mm-28", ActivityID: "act-2", Title: "Simplifying Expressions", ContentText: "Combine the laws in order: distribute, absorb, complement, then De Morgan. Example: A·B + A·¬B = A·(B + ¬B) = A·1 = A.", Order: 14,
				Question: "Simplify A·B + A·¬B.", Options: []string{"A·B", "A", "B", "1"}, CorrectIndex: 1,
				Explanation: "Factor A: A·(B + ¬B) = A·1 = A."},
			{ID: "mm-29", ActivityID: "act-2", Title: "From Algebra to Circuits", ContentText: "Every Boolean expression maps to a circuit of gates: AND for ·, OR for +, NOT for ¬. Simplifying the algebra simplifies the hardware — fewer gates, less power, fewer failures.", Order: 15,
				Question: "How many AND gates does A·B·C need?", Options: []string{"1", "2", "3", "4"}, CorrectIndex: 1,
				Explanation: "Two AND gates in series: (A·B)·C. A three-input AND can be built from two two-input ANDs."},
			{ID: "mm-30", ActivityID: "act-2", Title: "Real Machines, Real Logic", ContentText: "Your phone's ALU adds numbers with XOR and AND gates, checks passwords with AND chains, and stores memory with NAND latches. Boolean algebra runs the whole device.", Order: 16,
				Question: "Binary addition uses which gate for the sum bit?", Options: []string{"AND", "OR", "XOR", "NAND"}, CorrectIndex: 2,
				Explanation: "The sum bit of 1+1=0-with-carry is exactly XOR's truth table."},

			// ─── act-3: SEE Mathematics — Quadratic Equations (6 modules) ───
			{ID: "mm-31", ActivityID: "act-3", Title: "Standard Form", ContentText: "A quadratic equation has the form ax² + bx + c = 0, where a ≠ 0. The highest power of x is 2, which is why the graph is a parabola.", Order: 1,
				Question: "Which is a quadratic equation?", Options: []string{"x + 3 = 7", "x² − 5x + 6 = 0", "x³ − 2 = 0", "2x = 10"}, CorrectIndex: 1,
				Explanation: "A quadratic has an x² term as its highest power. x² − 5x + 6 = 0 fits ax² + bx + c = 0 with a=1."},
			{ID: "mm-32", ActivityID: "act-3", Title: "Finding Roots by Factoring", ContentText: "To factor x² − 5x + 6, find two numbers that multiply to 6 and add to −5: −2 and −3. So (x−2)(x−3) = 0, giving roots x=2 and x=3.", Order: 2,
				Question: "What are the roots of x² − 5x + 6 = 0?", Options: []string{"x = 2 and x = 3", "x = −2 and x = −3", "x = 1 and x = 6", "x = 5 and x = 6"}, CorrectIndex: 0,
				Explanation: "Factored, (x−2)(x−3)=0. A product is zero when one factor is zero, so x=2 or x=3. Good factoring work!"},
			{ID: "mm-33", ActivityID: "act-3", Title: "The Quadratic Formula", ContentText: "When factoring is tricky, use x = (−b ± √(b²−4ac)) / 2a. It always works — every quadratic has at most two roots.", Order: 3,
				Question: "For 2x² + 4x − 6 = 0, what is the value of b² − 4ac?", Options: []string{"16 − 48 = −32", "16 + 48 = 64", "4 − 48 = −44", "8 − 12 = −4"}, CorrectIndex: 1,
				Explanation: "a=2, b=4, c=−6. So b² − 4ac = 16 − 4(2)(−6) = 16 + 48 = 64. A perfect square — the roots will be whole numbers."},
			{ID: "mm-34", ActivityID: "act-3", Title: "The Discriminant", ContentText: "The discriminant b² − 4ac tells you how many roots exist: positive means two real roots, zero means one (repeated), negative means none in the real numbers.", Order: 4,
				Question: "A discriminant of −9 means the equation has…", Options: []string{"Two real roots", "One real root", "No real roots", "Infinitely many roots"}, CorrectIndex: 2,
				Explanation: "A negative discriminant means the square root is not a real number, so there are no real roots. Nothing to worry about — it's a property of the equation, not a mistake."},
			{ID: "mm-35", ActivityID: "act-3", Title: "Solving by Completing the Square", ContentText: "Rewrite x² + 6x = 7 by adding (6/2)² = 9 to both sides: (x+3)² = 16, then x+3 = ±4, so x = 1 or x = −7.", Order: 5,
				Question: "Completing the square turns x² + 6x = 7 into…", Options: []string{"(x+3)² = 7", "(x+3)² = 16", "(x+6)² = 16", "(x+3)² = 9"}, CorrectIndex: 1,
				Explanation: "Half of 6 is 3; squaring gives 9. Adding 9 to both sides yields (x+3)² = 7 + 9 = 16. Then x+3 = ±4."},
			{ID: "mm-36", ActivityID: "act-3", Title: "Quadratics in Word Problems", ContentText: "The area of a rectangle is length × width. If the length is 3 m more than the width and the area is 10 m², then w(w+3) = 10 → w² + 3w − 10 = 0 → (w+5)(w−2) = 0 → w = 2 m (width is positive).", Order: 6,
				Question: "A rectangle's width is 3 m less than its length, and its area is 10 m². What is the width?", Options: []string{"2 m", "3 m", "5 m", "10 m"}, CorrectIndex: 0,
				Explanation: "w(w+3)=10 gives w²+3w−10=0, which factors to (w+5)(w−2)=0. Lengths can't be negative, so w=2 m. A rectangle with width 2 and length 5 has area 10."},

			// ─── act-4: SEE Science — Electricity (6 modules) ───
			{ID: "mm-37", ActivityID: "act-4", Title: "Current, Voltage, Resistance", ContentText: "Current (I) is the flow of charge in amperes, voltage (V) is the push that drives it in volts, and resistance (R) is the opposition to flow in ohms.", Order: 1,
				Question: "What unit measures electric current?", Options: []string{"Volt", "Ohm", "Ampere", "Watt"}, CorrectIndex: 2,
				Explanation: "Current is measured in amperes (A). Volts measure voltage, ohms measure resistance, and watts measure power."},
			{ID: "mm-38", ActivityID: "act-4", Title: "Ohm's Law", ContentText: "Ohm's law says V = I × R. A 12 V battery pushing 3 A through a bulb means the bulb has resistance R = V/I = 4 Ω.", Order: 2,
				Question: "A 12 V battery drives 3 A of current. What is the resistance?", Options: []string{"4 Ω", "15 Ω", "36 Ω", "9 Ω"}, CorrectIndex: 0,
				Explanation: "R = V/I = 12/3 = 4 Ω. Dividing, not multiplying, is the easy slip here — you got it right."},
			{ID: "mm-39", ActivityID: "act-4", Title: "Series Circuits", ContentText: "In a series circuit, the same current flows through every component and resistances add up: R_total = R₁ + R₂ + …", Order: 3,
				Question: "Two resistors, 2 Ω and 3 Ω, are in series. What is the total resistance?", Options: []string{"1.2 Ω", "5 Ω", "6 Ω", "2.5 Ω"}, CorrectIndex: 1,
				Explanation: "In series, resistances add: 2 + 3 = 5 Ω. Same current flows through both."},
			{ID: "mm-40", ActivityID: "act-4", Title: "Parallel Circuits", ContentText: "In a parallel circuit, each branch gets the full voltage, and the total current is the sum of the branch currents. The total resistance is smaller than the smallest branch.", Order: 4,
				Question: "In a parallel circuit, the voltage across each branch is…", Options: []string{"Divided equally", "The same as the source voltage", "Always zero", "Half the source voltage"}, CorrectIndex: 1,
				Explanation: "Parallel branches each receive the full source voltage. That's why home appliances in Nepal run on the same 220 V mains."},
			{ID: "mm-41", ActivityID: "act-4", Title: "Electrical Power", ContentText: "Power P = V × I in watts. A 220 V device drawing 2 A uses P = 440 W. This is how electricity bills are calculated.", Order: 5,
				Question: "A heater on 220 V draws 5 A. What is its power?", Options: []string{"44 W", "1100 W", "225 W", "110 W"}, CorrectIndex: 1,
				Explanation: "P = V × I = 220 × 5 = 1100 W. One kilowatt of heating — that's why heaters are energy-hungry."},
			{ID: "mm-42", ActivityID: "act-4", Title: "Fuses and Safety", ContentText: "A fuse is a thin wire that melts when current exceeds its rating, breaking the circuit before wires overheat. It protects the device and prevents fires.", Order: 6,
				Question: "What is the main job of a fuse?", Options: []string{"Increase voltage", "Save electricity", "Break the circuit on excess current", "Store charge"}, CorrectIndex: 2,
				Explanation: "A fuse deliberately melts on excess current, cutting power before overheating can start a fire. Safety first, always."},

			// ─── act-5: SEE English — Tenses (6 modules) ───
			{ID: "mm-43", ActivityID: "act-5", Title: "Present Simple", ContentText: "Present simple states facts and habits: \"She walks to school every day.\" Use the base verb; add -s/-es for he/she/it.", Order: 1,
				Question: "Which is correct?", Options: []string{"She walk to school.", "She walks to school.", "She walking to school.", "She walked to school."}, CorrectIndex: 1,
				Explanation: "With he/she/it, present simple adds -s: \"She walks to school.\" Habitual actions use present simple."},
			{ID: "mm-44", ActivityID: "act-5", Title: "Present Continuous", ContentText: "Present continuous describes actions happening now: \"I am studying for the SEE.\" Form: am/is/are + verb-ing.", Order: 2,
				Question: "Which is correct?", Options: []string{"I am study now.", "I is studying now.", "I am studying now.", "I studying now."}, CorrectIndex: 2,
				Explanation: "Use am/is/are + the -ing form: \"I am studying now.\" Present continuous is for actions in progress."},
			{ID: "mm-45", ActivityID: "act-5", Title: "Past Simple", ContentText: "Past simple describes finished actions: \"He finished the exam at noon.\" Regular verbs add -ed; irregular verbs change form (go → went).", Order: 3,
				Question: "Which is the correct past simple?", Options: []string{"He goed to school.", "He went to school.", "He go to school.", "He has go to school."}, CorrectIndex: 1,
				Explanation: "\"Go\" is irregular: go → went. \"Goed\" is a common slip — remember the irregular list and you'll do great."},
			{ID: "mm-46", ActivityID: "act-5", Title: "Present Perfect", ContentText: "Present perfect links past to now: \"I have finished my homework.\" Form: have/has + past participle. It shows a result that matters today.", Order: 4,
				Question: "Which is correct?", Options: []string{"I have finish my homework.", "I have finished my homework.", "I finished my homework yesterday", "I finish my homework."}, CorrectIndex: 1,
				Explanation: "Present perfect = have/has + past participle: \"I have finished my homework.\" The past participle of finish is finished."},
			{ID: "mm-47", ActivityID: "act-5", Title: "Future with Will", ContentText: "\"Will\" expresses future intentions and predictions: \"The bus will arrive at 8.\" Form: will + base verb.", Order: 5,
				Question: "Which is correct?", Options: []string{"We will arrives tomorrow.", "We will arrive tomorrow.", "We will arriving tomorrow.", "We will to arrive tomorrow."}, CorrectIndex: 1,
				Explanation: "Will is followed by the base verb: \"We will arrive tomorrow.\" No -s, no -ing after will."},
			{ID: "mm-48", ActivityID: "act-5", Title: "Choosing the Right Tense", ContentText: "Ask: is it a habit (simple), happening now (continuous), finished (past), or a result that matters now (perfect)? The time word — every day, now, yesterday, already — is the biggest clue.", Order: 6,
				Question: "\"She ___ already eaten.\" Which word completes the sentence?", Options: []string{"have", "has", "is", "was"}, CorrectIndex: 1,
				Explanation: "\"Already\" signals present perfect, and with she we use has: \"She has already eaten.\""},
		}
		for _, m := range microModules {
			DB.Create(&m)
		}
	}

	// Seed Courses
	var courseCount int64
	DB.Model(&domain.Course{}).Count(&courseCount)
	if courseCount == 0 {
		courses := []domain.Course{
			{ID: "course-1", Title: "Fundamentals of Logic & Gates", Category: "Computer Science", Difficulty: "Beginner", Duration: "3 hours", Rating: 4.9},
			{ID: "course-2", Title: "Boolean Algebra & Truth Tables", Category: "Computer Science", Difficulty: "Intermediate", Duration: "4 hours", Rating: 4.8},
			{ID: "course-3", Title: "Data Structures & Offline Caching", Category: "Backend", Difficulty: "Advanced", Duration: "6 hours", Rating: 4.9},
			{ID: "course-4", Title: "Modern Frontend & Micro-Animations", Category: "Frontend", Difficulty: "Intermediate", Duration: "5 hours", Rating: 4.7},
			{ID: "course-5", Title: "UI/UX Accessibility for Low-Bandwidth", Category: "Design", Difficulty: "Beginner", Duration: "2.5 hours", Rating: 5.0},
		}
		for _, c := range courses {
			DB.Create(&c)
		}
	}

	// WP-0.2 C5: the Enrolled column is now a derived count. Recompute it from
	// the enrollments table at every startup so any legacy seed value (or a
	// direct DB edit) can never serve fabricated numbers.
	DB.Exec("UPDATE courses SET enrolled = (SELECT COUNT(*) FROM enrollments WHERE enrollments.course_id = courses.id)")

	// Seed real enrollments for the demo learner so the catalog and dashboard
	// derive honest, non-zero numbers from actual rows.
	var enrCount int64
	DB.Model(&domain.Enrollment{}).Count(&enrCount)
	if enrCount == 0 {
		for _, courseID := range []string{"course-1", "course-3"} {
			DB.Create(&domain.Enrollment{ID: "enr-" + courseID, UserID: "user-123", CourseID: courseID, CreatedAt: time.Now()})
		}
		DB.Exec("UPDATE courses SET enrolled = (SELECT COUNT(*) FROM enrollments WHERE enrollments.course_id = courses.id)")
	}

	// Seed Daily Activity
	var actCount int64
	DB.Model(&domain.DailyActivity{}).Count(&actCount)
	if actCount == 0 {
		dailyActivities := []domain.DailyActivity{
			{ID: "da-1", LearnerID: "user-123", DayName: "Mon", Score: 65, Duration: 20},
			{ID: "da-2", LearnerID: "user-123", DayName: "Tue", Score: 70, Duration: 25},
			{ID: "da-3", LearnerID: "user-123", DayName: "Wed", Score: 68, Duration: 15},
			{ID: "da-4", LearnerID: "user-123", DayName: "Thu", Score: 75, Duration: 30},
			{ID: "da-5", LearnerID: "user-123", DayName: "Fri", Score: 85, Duration: 45},
			{ID: "da-6", LearnerID: "user-123", DayName: "Sat", Score: 82, Duration: 40},
			{ID: "da-7", LearnerID: "user-123", DayName: "Sun", Score: 88, Duration: 50},
		}
		for _, da := range dailyActivities {
			DB.Create(&da)
		}
	}

	// Seed school classes independently of users so existing databases also
	// receive a working demo class (teacher mod-1, student user-123).
	var classCount int64
	DB.Model(&domain.Class{}).Count(&classCount)
	if classCount == 0 {
		DB.Create(&domain.Class{ID: "cls-1", Name: "Grade 10 A", Grade: "10", Section: "A", TeacherID: "mod-1", InviteCode: "LOG101", CreatedAt: time.Now()})
		DB.Create(&domain.ClassMember{ClassID: "cls-1", UserID: "user-123", JoinedAt: time.Now()})
	}

	// Seed one welcome announcement for demo purposes.
	var annCount int64
	DB.Model(&domain.Announcement{}).Count(&annCount)
	if annCount == 0 {
		DB.Create(&domain.Announcement{
			ID: "ann-1", Title: "Welcome to LOG",
			Body:     "Learn anytime, anywhere — your progress saves even without internet. Happy learning!",
			AuthorID: "admin-1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
	}

	// Phase 3 content (WP-3.1 OER metadata + packs, WP-3.2 SEE units).
	seedPhase3()
}
