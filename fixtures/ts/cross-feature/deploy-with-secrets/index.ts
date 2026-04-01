import { secrets, output } from "kit";

// Read a secret during deployment init
const key = secrets.get("NONEXISTENT_SECRET_XYZ");
output({ secretValue: key, isEmpty: key === "" });
