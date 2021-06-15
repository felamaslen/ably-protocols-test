import { v4 as uuidv4 } from "uuid";

import { runStateful } from "./stateful";
import { runStateless } from "./stateless";

async function run(): Promise<void> {
  switch (process.argv[2]) {
    case "--stateless":
      await runStateless(3);
      break;
    case "--stateful":
      await runStateful(10, uuidv4());
      break;
    default:
      throw new Error("Usage: ts-node <script> --stateless | --stateful");
  }
}

if (require.main === module) {
  run()
    .then(() => {
      console.log("Done");
      process.exit(0);
    })
    .catch((err) => {
      console.error("Unhandled error:", err);
      process.exit(1);
    });
}
