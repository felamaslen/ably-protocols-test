import md5 from "md5";
import net from "net";

import { MAX_RETRIES } from "./constants";
import { calculateExponentialDelay } from "./utils";

export async function runStateful(
  port: number,
  n: number,
  uuid: string
): Promise<void> {
  console.log("Running stateful client");

  const sequence: number[] = [];
  let expectedChecksum: string = "";

  let numRetries = 0;

  const initiateConnection = (m = 0): Promise<void> =>
    new Promise((resolve, reject) => {
      const client = new net.Socket();

      let numReceived = 0;

      client.connect(port, "127.0.0.1", () => {
        console.log("Connected");
        client.write(
          `Y${uuid}${String(n).padStart(5, "0")}${String(m).padStart(5, "0")}`
        );
      });

      client.on("connect", () => {
        numRetries = 0;
      });

      client.on("data", (data) => {
        const messages = data.toString("utf8").split("\n");

        const numberMessages = messages.filter((msg) => /^\d+$/.test(msg));
        numberMessages.forEach((num) => {
          console.debug(new Date(), "received", num);
          sequence.push(Number(num));
        });

        numReceived += numberMessages.length;

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

      let retried = false;

      const onError = (err?: Error): void => {
        console.warn("Handling error", err);

        if (numRetries > MAX_RETRIES) {
          reject("ERR_MAX_RETRIES");
        } else if (!retried) {
          numRetries++;
          retried = true;
          const exponentialDelayMs = calculateExponentialDelay(numRetries);

          console.log("Waiting", exponentialDelayMs, "ms");
          setTimeout(async () => {
            // Here, we resume the sequence with a new connection
            await initiateConnection(m + numReceived);
            resolve();
          }, exponentialDelayMs);
        }
      };

      client.on("close", () => {
        onError();
      });

      client.on("error", (err) => {
        onError(err);
      });
    });

  await initiateConnection();

  const receivedChecksum = md5(JSON.stringify(sequence));

  console.log("Result found", {
    receivedChecksum,
    passedCheck: receivedChecksum === expectedChecksum,
  });
}
