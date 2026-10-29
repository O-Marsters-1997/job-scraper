import { z } from "zod";

export const dimensionKeySchema = z.enum([
	"tech",
	"role",
	"domain",
	"seniority",
	"work",
	"stage",
	"size",
	"employment",
]);
