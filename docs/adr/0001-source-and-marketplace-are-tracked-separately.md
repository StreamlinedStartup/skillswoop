# Sources and marketplaces are tracked separately, even for the same repo

A repo can hold skills, plugins, or both. When it holds both, swoop remembers it once as a Source (skills section) and once as a Marketplace (plugins section), and the two are independent: removing it from one leaves the other. We chose this over one merged "repo" entry because people manage skills and plugins as separate jobs, and a removal in one section should never surprise them in the other.
