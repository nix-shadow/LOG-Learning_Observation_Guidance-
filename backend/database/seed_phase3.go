package database

import (
	"log-backend/internal/domain"
)

// seedPhase3 grows the catalog independently of user rows so existing
// databases receive the Phase 3 content on next startup:
//
//   - WP-3.1: backfills honest OER metadata on every existing activity and
//     imports two OER packs (original LOG-authored content shared under
//     CC BY-SA 4.0 — real licenses, real attribution).
//   - WP-3.2: seeds full SEE 9–12 subject units with practice-first quiz
//     banks (original content with a supportive explanation per question).
//
// Every module block is gated per activity, so an existing database that
// already has one subject's modules is never re-seeded or duplicated.
func seedPhase3() {
	// ─── WP-3.1: honest OER metadata backfill ───
	// Existing demo activities are original LOG-authored content — they are
	// labeled "Own work (LOG team)", never falsely attributed to a third party.
	ownWork := []string{"act-1", "act-2", "act-3", "act-4", "act-5"}
	for _, id := range ownWork {
		DB.Model(&domain.Activity{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"license":     "Own work (LOG team)",
				"license_url": "",
				"attribution": "LOG Learning Team",
			})
	}

	// ─── WP-3.4 (RC-12): NSL caption metadata ───
	// act-4 carries an honest caption track: a written NSL caption for its
	// lesson video (the video itself is sourced with consent at deployment —
	// see docs/PARTNERSHIP_LANES.md). Other activities have none, and the UI
	// shows nothing rather than inventing a caption.
	DB.Model(&domain.Activity{}).Where("id = ?", "act-4").Updates(map[string]interface{}{
		"caption_text": "विद्युत् प्रवाह र परिपथ — नेपाली सांकेतिक भाषामा व्याख्या। चाल, भोल्टेज र प्रतिरोधका अवधारणा साङ्केतिक भाषामा देखाइएको छ। (Electric current and circuits — explained in Nepali Sign Language: current, voltage, and resistance.)",
	})

	// ─── WP-3.2: SEE subject units ───
	// act-6 SEE Mathematics: Statistics
	seedModulesFor("act-6", []domain.MicroModule{
		{ID: "mm-49", ActivityID: "act-6", Title: "The Mean", ContentText: "The mean is the sum of all values divided by how many there are. For scores 6, 8, and 10: (6+8+10)/3 = 24/3 = 8.", Order: 1,
			Question: "What is the mean of 4, 6, and 8?", Options: []string{"5", "6", "7", "8"}, CorrectIndex: 1,
			Explanation: "Sum = 4+6+8 = 18, and 18/3 = 6. The mean is the fair share of the total."},
		{ID: "mm-50", ActivityID: "act-6", Title: "The Median", ContentText: "The median is the middle value when data is sorted. For 3, 5, 9 the median is 5. With an even count, take the mean of the two middle values.", Order: 2,
			Question: "What is the median of 2, 4, 7, 9?", Options: []string{"4", "5.5", "7", "2"}, CorrectIndex: 1,
			Explanation: "Sorted values are 2, 4, 7, 9 — two middle values 4 and 7 give (4+7)/2 = 5.5."},
		{ID: "mm-51", ActivityID: "act-6", Title: "The Mode", ContentText: "The mode is the value that appears most often. In 2, 3, 3, 5, 3, 7 the mode is 3. A data set can have more than one mode or none.", Order: 3,
			Question: "What is the mode of 1, 4, 4, 6, 4, 9?", Options: []string{"1", "4", "6", "9"}, CorrectIndex: 1,
			Explanation: "4 appears three times — more than any other value, so 4 is the mode."},
		{ID: "mm-52", ActivityID: "act-6", Title: "The Range", ContentText: "The range is the difference between the largest and smallest values. It shows how spread out the data is. For 3, 8, 12: range = 12 − 3 = 9.", Order: 4,
			Question: "What is the range of 5, 9, 12, 18?", Options: []string{"7", "13", "18", "5"}, CorrectIndex: 1,
			Explanation: "Largest is 18, smallest is 5, so the range is 18 − 5 = 13."},
		{ID: "mm-53", ActivityID: "act-6", Title: "Statistics in Daily Life", ContentText: "Newspapers, teachers, and health workers all use mean, median, mode, and range to summarize data. The median is useful when one extreme value would skew the mean.", Order: 5,
			Question: "Five test scores are 10, 90, 95, 96, 98. Which measure best summarizes the class?", Options: []string{"Mean", "Median", "Mode", "Range"}, CorrectIndex: 1,
			Explanation: "The extreme 10 pulls the mean down unfairly; the median (95) represents the typical score far better."},
	})

	// act-7 SEE Science: Chemical Reactions
	seedModulesFor("act-7", []domain.MicroModule{
		{ID: "mm-54", ActivityID: "act-7", Title: "Reactants and Products", ContentText: "In a chemical reaction, the substances you start with are reactants and the new substances formed are products. A reaction arrow → shows the change.", Order: 1,
			Question: "In the reaction A + B → C, what is C?", Options: []string{"A reactant", "A product", "A catalyst", "A solvent"}, CorrectIndex: 1,
			Explanation: "Everything on the right of the arrow is a product — C is what the reaction makes."},
		{ID: "mm-55", ActivityID: "act-7", Title: "Balancing Equations", ContentText: "A balanced equation has the same number of each atom on both sides. Mass is conserved — atoms are rearranged, never created or destroyed.", Order: 2,
			Question: "Which equation is balanced? H₂ + O₂ → H₂O", Options: []string{"2H₂ + O₂ → 2H₂O", "H₂ + 2O₂ → H₂O", "2H₂ + O₂ → H₂O", "H₂ + O₂ → 2H₂O"}, CorrectIndex: 0,
			Explanation: "Left: 4 H and 2 O. Right: 4 H and 2 O. Every atom matches — that's a balanced equation."},
		{ID: "mm-56", ActivityID: "act-7", Title: "Types of Reactions", ContentText: "Common reaction types include combination (two substances join), decomposition (one substance splits), and displacement (an element swaps places).", Order: 3,
			Question: "2H₂O → 2H₂ + O₂ is an example of…", Options: []string{"Combination", "Decomposition", "Neutralization", "Precipitation"}, CorrectIndex: 1,
			Explanation: "One substance (water) splits into two — that's decomposition."},
		{ID: "mm-57", ActivityID: "act-7", Title: "Acids, Bases, and Neutralization", ContentText: "Acids taste sour and turn litmus red; bases taste bitter and turn it blue. When an acid meets a base they neutralize, forming salt and water.", Order: 4,
			Question: "Acid + Base → ?", Options: []string{"Salt + Water", "More acid", "Pure metal", "Carbon dioxide only"}, CorrectIndex: 0,
			Explanation: "Neutralization produces a salt and water — that's why antacids calm stomach acid."},
		{ID: "mm-58", ActivityID: "act-7", Title: "Reactions Around Us", ContentText: "Rusting iron, baking bread, and burning wood are all chemical reactions. Recognizing them helps us design safer, cleaner processes.", Order: 5,
			Question: "Which everyday event is a chemical reaction?", Options: []string{"Melting ice", "Cutting paper", "Rusting of iron", "Boiling water"}, CorrectIndex: 2,
			Explanation: "Rusting changes iron into a new substance (iron oxide) — a chemical reaction. The others are physical changes."},
	})

	// act-8 SEE Social Studies: Our Federal Republic
	seedModulesFor("act-8", []domain.MicroModule{
		{ID: "mm-59", ActivityID: "act-8", Title: "The Federal Structure", ContentText: "Nepal is a federal democratic republic. Power is shared across three levels of government: federal, provincial, and local — each with its own responsibilities.", Order: 1,
			Question: "How many levels of government does Nepal's federal structure have?", Options: []string{"One", "Two", "Three", "Four"}, CorrectIndex: 2,
			Explanation: "Federal, provincial, and local — three levels that work together to serve citizens."},
		{ID: "mm-60", ActivityID: "act-8", Title: "The Provinces", ContentText: "Nepal has seven provinces, each with its own assembly and chief minister. Provinces coordinate with the center on shared matters like education and health.", Order: 2,
			Question: "How many provinces does Nepal have?", Options: []string{"Five", "Six", "Seven", "Nine"}, CorrectIndex: 2,
			Explanation: "Nepal has seven provinces, formed under the 2072 constitution."},
		{ID: "mm-61", ActivityID: "act-8", Title: "The Constitution of Nepal 2072", ContentText: "The constitution, adopted in 2072 BS, declares Nepal a federal democratic republic and guarantees fundamental rights including education and equality.", Order: 3,
			Question: "In which year was the current Constitution of Nepal promulgated?", Options: []string{"2063 BS", "2072 BS", "2069 BS", "2080 BS"}, CorrectIndex: 1,
			Explanation: "The constitution was promulgated on 2072-03-03 BS (16 September 2015)."},
		{ID: "mm-62", ActivityID: "act-8", Title: "Local Government", ContentText: "Local governments — rural municipalities, municipalities, and metropolitan cities — deliver services closest to citizens: schools, health posts, and local roads.", Order: 4,
			Question: "Which level is closest to citizens?", Options: []string{"Federal", "Provincial", "Local", "International"}, CorrectIndex: 2,
			Explanation: "Local government works directly with communities on schools, health, and local infrastructure."},
		{ID: "mm-63", ActivityID: "act-8", Title: "Civic Duties", ContentText: "With rights come duties: paying taxes, voting, keeping public property clean, and respecting others. Active citizens make democracy stronger.", Order: 5,
			Question: "Which is a civic duty of every citizen?", Options: []string{"Paying taxes", "Ignoring elections", "Littering roads", "Skipping school"}, CorrectIndex: 0,
			Explanation: "Paying taxes funds schools, roads, and hospitals that everyone shares."},
	})

	// act-9 SEE Computer Science: Number Systems
	seedModulesFor("act-9", []domain.MicroModule{
		{ID: "mm-64", ActivityID: "act-9", Title: "Why Binary?", ContentText: "Computers use binary — ones and zeros — because a switch (or transistor) is either on or off. Two states are enough to represent any number.", Order: 1,
			Question: "Which number system do computers use at the hardware level?", Options: []string{"Decimal", "Binary", "Roman", "Vigesimal"}, CorrectIndex: 1,
			Explanation: "Circuits are on/off, so binary (base 2) is the natural language of hardware."},
		{ID: "mm-65", ActivityID: "act-9", Title: "Decimal to Binary", ContentText: "To convert 13 to binary, repeatedly divide by 2 and keep the remainders: 13 → 1101. Read the remainders bottom to top.", Order: 2,
			Question: "What is 13 in binary?", Options: []string{"1101", "1011", "1110", "1001"}, CorrectIndex: 0,
			Explanation: "13 = 8 + 4 + 1 = 1101₂. Starting from the 8's place: 1, 1, 0, 1."},
		{ID: "mm-66", ActivityID: "act-9", Title: "Bits and Bytes", ContentText: "One binary digit is a bit; eight bits make a byte. A byte can represent 2⁸ = 256 different values — enough for one character.", Order: 3,
			Question: "How many bits are in one byte?", Options: []string{"4", "8", "16", "32"}, CorrectIndex: 1,
			Explanation: "Eight bits make a byte, and 2⁸ = 256 values fit in it."},
		{ID: "mm-67", ActivityID: "act-9", Title: "Hexadecimal", ContentText: "Hexadecimal (base 16) uses digits 0–9 and letters A–F. It's a compact way to write binary — one hex digit equals four bits.", Order: 4,
			Question: "Which hex digit equals binary 1111?", Options: []string{"E", "F", "9", "A"}, CorrectIndex: 1,
			Explanation: "1111₂ = 15 in decimal, which is F in hexadecimal."},
		{ID: "mm-68", ActivityID: "act-9", Title: "Where You See These Systems", ContentText: "MAC addresses are written in hex, and error codes and colors (like #FF5733) use hex too. Learning number systems helps you read the tech around you.", Order: 5,
			Question: "Web colors like #FF5733 are written in which system?", Options: []string{"Binary", "Hexadecimal", "Octal", "Decimal"}, CorrectIndex: 1,
			Explanation: "CSS colors use six hex digits — two each for red, green, and blue."},
	})

	// act-10 SEE Nepali: व्याकरण — bilingual, original LOG-authored content.
	seedModulesFor("act-10", []domain.MicroModule{
		{ID: "mm-69", ActivityID: "act-10", Title: "संज्ञा (Noun)", ContentText: "संज्ञा भनेकै कुनै व्यक्ति, वस्तु, ठाउँ वा भावको नाम हो। जस्तै: राम, काठमाडौं, किताब, खुशी।", Order: 1,
			Question: "तलका मध्ये कुन शब्द संज्ञा हो?", Options: []string{"दौडिनु", "सुन्दर", "विद्यालय", "छिटो"}, CorrectIndex: 2,
			Explanation: "'विद्यालय' ठाउँ/वस्तुको नाम हो, त्यसैले यो संज्ञा हो।"},
		{ID: "mm-70", ActivityID: "act-10", Title: "सर्वनाम (Pronoun)", ContentText: "सर्वनामले संज्ञाको ठाउँमा प्रयोग हुने शब्दलाई जनाउँछ। जस्तै: म, तिमी, उनी, हामी।", Order: 2,
			Question: "तलका मध्ये कुन शब्द सर्वनाम हो?", Options: []string{"पुस्तक", "हामी", "घर", "हरियो"}, CorrectIndex: 1,
			Explanation: "'हामी' ले व्यक्तिको नामको ठाउँमा प्रयोग हुने शब्द हो, त्यसैले यो सर्वनाम हो।"},
		{ID: "mm-71", ActivityID: "act-10", Title: "क्रिया (Verb)", ContentText: "क्रिया भनेको गर्ने कामलाई जनाउने शब्द हो। जस्तै: पढ्नु, खानु, दौडनु, बोल्नु।", Order: 3,
			Question: "तलका मध्ये कुन शब्द क्रिया हो?", Options: []string{"सुन्दरता", "लेख्नु", "पानी", "रातो"}, CorrectIndex: 1,
			Explanation: "'लेख्नु' ले कामलाई जनाउँछ, त्यसैले यो क्रिया हो।"},
		{ID: "mm-72", ActivityID: "act-10", Title: "वचन (Number)", ContentText: "वचनले एक वा धेरैलाई जनाउँछ। एकवचन: किताब, बहुवचन: किताबहरू। अन्त्यमा '-हरू' थप्दा बहुवचन बन्छ।", Order: 4,
			Question: "'केटीहरू' कुन वचनमा छ?", Options: []string{"एकवचन", "बहुवचन", "दुवै होइन", "क्रिया"}, CorrectIndex: 1,
			Explanation: "'-हरू' जोडिएकाले 'केटीहरू' बहुवचन हो।"},
	})

	// ─── WP-3.1: two OER packs — original LOG-authored content shared under
	// a real, machine-readable CC BY-SA 4.0 license with honest attribution.
	seedModulesFor("act-11", []domain.MicroModule{
		{ID: "mm-73", ActivityID: "act-11", Title: "Probability Basics", ContentText: "Probability measures how likely an event is: P(event) = favourable outcomes ÷ total outcomes. It always lies between 0 and 1.", Order: 1,
			Question: "What is the probability of a fair coin landing heads?", Options: []string{"0", "1/2", "1", "2"}, CorrectIndex: 1,
			Explanation: "One favourable outcome out of two equally likely ones: 1/2."},
		{ID: "mm-74", ActivityID: "act-11", Title: "Sample Space", ContentText: "The sample space is the set of every possible outcome. Rolling a die gives {1, 2, 3, 4, 5, 6} — six equally likely outcomes.", Order: 2,
			Question: "How many outcomes are in the sample space of a single die?", Options: []string{"2", "4", "6", "12"}, CorrectIndex: 2,
			Explanation: "A standard die has six faces, so the sample space has six outcomes."},
		{ID: "mm-75", ActivityID: "act-11", Title: "Complementary Events", ContentText: "The complement of an event is everything NOT in it. Their probabilities add to 1: P(A) + P(not A) = 1. If P(rain) = 0.3, P(no rain) = 0.7.", Order: 3,
			Question: "If P(rain) = 0.3, what is P(no rain)?", Options: []string{"0.3", "0.5", "0.7", "1.0"}, CorrectIndex: 2,
			Explanation: "Rain and no rain are complements, so 1 − 0.3 = 0.7."},
		{ID: "mm-76", ActivityID: "act-11", Title: "Probability in Daily Life", ContentText: "Forecasters, insurers, and doctors all use probability to make careful decisions under uncertainty.", Order: 4,
			Question: "A bag has 3 red and 5 blue marbles. Picking a blue marble has probability…", Options: []string{"3/8", "5/8", "1/2", "5/3"}, CorrectIndex: 1,
			Explanation: "Five blue marbles out of eight total: 5/8."},
	})

	seedModulesFor("act-12", []domain.MicroModule{
		{ID: "mm-77", ActivityID: "act-12", Title: "What is an Ecosystem?", ContentText: "An ecosystem is a community of living things interacting with their environment. A forest, a pond, and a rice field are all ecosystems.", Order: 1,
			Question: "Which is an ecosystem?", Options: []string{"A single tree", "A pond with its plants and fish", "A rock", "A cloud"}, CorrectIndex: 1,
			Explanation: "An ecosystem includes living things AND their surroundings working together — a pond fits."},
		{ID: "mm-78", ActivityID: "act-12", Title: "Food Chains", ContentText: "A food chain shows who eats whom: grass → goat → human. Energy flows from plants (producers) up through consumers.", Order: 2,
			Question: "In grass → goat → human, which is the producer?", Options: []string{"Grass", "Goat", "Human", "All of them"}, CorrectIndex: 0,
			Explanation: "Producers make their own food — grass uses sunlight. Goats and humans are consumers."},
		{ID: "mm-79", ActivityID: "act-12", Title: "Pollution", ContentText: "Pollution harms ecosystems: plastic in rivers, smoke in the air, and chemicals in soil. Clean water, air, and soil are shared resources.", Order: 3,
			Question: "Which of these directly harms a river ecosystem?", Options: []string{"Planting trees", "Plastic waste", "Reducing smoke", "Recycling"}, CorrectIndex: 1,
			Explanation: "Plastic waste clogs rivers and harms the animals that live there — a direct harm."},
		{ID: "mm-80", ActivityID: "act-12", Title: "Conservation", ContentText: "Conservation protects nature for the future: planting trees, saving water, protecting wildlife, and reducing waste. Small daily habits add up.", Order: 4,
			Question: "Which habit supports conservation?", Options: []string{"Using single-use plastic", "Wasting water", "Planting a tree", "Burning waste"}, CorrectIndex: 2,
			Explanation: "Planting trees restores habitat and cleans the air — a conservation win."},
	})

	// OER pack metadata lives on the Activity rows themselves. Insert the two
	// packs (skipping ids that already exist) so the license + attribution
	// fields are real rows, not just labels.
	for _, act := range []domain.Activity{
		{
			ID: "act-11", Title: "SEE Mathematics: Probability (OER Pack)", Description: "Probability, sample spaces, and complements — a practice-first pack.", Topic: "Mathematics", Order: 11,
			Difficulty: "Beginner",
			License:    "CC BY-SA 4.0", Attribution: "LOG Learning Team (original content)", SourceURL: "https://github.com/log/learning-content",
		},
		{
			ID: "act-12", Title: "SEE Science: Our Environment (OER Pack)", Description: "Ecosystems, food chains, and conservation — a practice-first pack.", Topic: "Science", Order: 12,
			Difficulty: "Beginner",
			License:    "CC BY-SA 4.0", Attribution: "LOG Learning Team (original content)", SourceURL: "https://github.com/log/learning-content",
		},
	} {
		var n int64
		DB.Model(&domain.Activity{}).Where("id = ?", act.ID).Count(&n)
		if n > 0 {
			continue
		}
		DB.Create(&act)
	}

	// New SEE activities (act-6..act-10) also need their Activity rows.
	for _, act := range []domain.Activity{
		{ID: "act-6", Title: "SEE Mathematics: Statistics", Description: "Mean, median, mode, and range with supportive practice.", Topic: "Mathematics", Order: 6, Difficulty: "Beginner"},
		{ID: "act-7", Title: "SEE Science: Chemical Reactions", Description: "Balancing equations, reaction types, and acids & bases.", Topic: "Science", Order: 7, Difficulty: "Beginner"},
		{ID: "act-8", Title: "SEE Social Studies: Our Federal Republic", Description: "Federal structure, provinces, and civic duties.", Topic: "Social Studies", Order: 8, Difficulty: "Beginner"},
		{ID: "act-9", Title: "SEE Computer Science: Number Systems", Description: "Binary, bytes, and hexadecimal with practice.", Topic: "Computer Science", Order: 9, Difficulty: "Beginner"},
		{ID: "act-10", Title: "SEE Nepali: व्याकरण", Description: "संज्ञा, सर्वनाम, क्रिया र वचन — अभ्याससहित।", Topic: "Nepali", Order: 10, Difficulty: "Beginner"},
	} {
		var n int64
		DB.Model(&domain.Activity{}).Where("id = ?", act.ID).Count(&n)
		if n > 0 {
			continue
		}
		act.License = "Own work (LOG team)"
		act.Attribution = "LOG Learning Team"
		DB.Create(&act)
	}
}

// seedModulesFor creates the given modules only when the activity has none
// yet, so restarting an existing database never duplicates content.
func seedModulesFor(activityID string, modules []domain.MicroModule) {
	var n int64
	DB.Model(&domain.MicroModule{}).Where("activity_id = ?", activityID).Count(&n)
	if n > 0 {
		return
	}
	for _, m := range modules {
		DB.Create(&m)
	}
}