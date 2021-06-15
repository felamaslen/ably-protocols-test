import md5 from "md5";
import net from "net";

import { MAX_RETRIES } from "./constants";
import { calculateExponentialDelay } from "./utils";

export async function runStateful(n: number, uuid: string): Promise<void> {
  console.log("Running stateful client");

  const sequence: number[] = [];
  let expectedChecksum: string = "";

  const initiateConnection = (numRetries = 0): Promise<void> =>
    new Promise((resolve, reject) => {
      const client = new net.Socket();

      client.connect(8081, "127.0.0.1", () => {
        console.log("Connected");
        client.write(`${String(n).padStart(5, "0")}${uuid}`);
      });

      client.on("data", (data) => {
        const messages = data.toString("utf8").split("\n");

        const numberMessages = messages.filter((msg) => /^\d+$/.test(msg));
        numberMessages.forEach((num) => {
          sequence.push(Number(num));
        });

        const checksumMessage = messages.find((msg) =>
          /^checksum=(\w+)$/.test(msg)
        );
        if (checksumMessage) {
          expectedChecksum = checksumMessage.split("=")[1];
        }

        if (messages.some((msg) => msg === "EOF")) {
          console.debug("received EOF; ending connection");
          client.end();
          resolve();
        }
      });

      client.on("close", () => {
        reject("Server closed connection unexpectedly");
      });

      client.on("error", (err) => {
        console.warn("Handling error", err);
        numRetries++;

        if (numRetries > MAX_RETRIES) {
          reject("ERR_MAX_RETRIES");
        } else {
          const exponentialDelayMs = calculateExponentialDelay(numRetries);

          console.log("Waiting", exponentialDelayMs, "ms");
          setTimeout(async () => {
            // Here, we resume the sequence with a new connection
            await initiateConnection(numRetries);
          }, exponentialDelayMs);
        }
      });
    });

  await initiateConnection();

  const receivedChecksum = md5(JSON.stringify(sequence));

  console.log("Result found", {
    receivedChecksum,
    passedCheck: receivedChecksum === expectedChecksum,
  });
}
