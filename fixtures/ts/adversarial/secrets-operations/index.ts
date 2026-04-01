import { secrets, output } from "kit";

// secrets.get for nonexistent key returns ""
const empty = secrets.get("ADVERSARIAL_NONEXISTENT_KEY");
output({ emptyResult: empty, isEmpty: empty === "" });
