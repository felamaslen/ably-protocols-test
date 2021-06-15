import net from "net";
import { MAX_RETRIES } from "./constants";
import { calculateExponentialDelay } from "./utils";

export async function runStateless(nStart: number): Promise<void> {
  console.log("Running stateless client");

  const initiateConnection = (
    a = 0,
    n = nStart,
    m = 0,
    numRetries = 0
  ): Promise<number> =>
    new Promise((resolve, reject) => {
      let sum = 0;
      let initialNumber = a;
      let numReceived = 0;

      const client = new net.Socket();

      client.connect(8080, "127.0.0.1", () => {
        console.log("Connected; writing", a, n, m);
        client.write(
          `${String(a).padStart(3, "0")}${String(n).padStart(5, "0")}${String(
            m
          ).padStart(5, "0")}`
        );
      });

      client.on("data", (data) => {
        const messages = data.toString("utf8").split("\n");

        const numbers = messages.filter((msg) => /^\d+$/.test(msg));

        sum = numbers.reduce<number>((last, num) => last + Number(num), sum);
        if (!initialNumber) {
          initialNumber = Number(numbers[0]);
        }

        numReceived += numbers.length;

        numbers.forEach((num) => {
          console.debug(new Date(), "received", num);
        });

        if (messages.some((msg) => msg === "EOF")) {
          console.debug("received EOF; ending connection");
          client.end();
          resolve(sum);
        }
      });

      const onError = (err?: Error): void => {
        console.warn("Handling error", err);
        numRetries++;

        if (numRetries > MAX_RETRIES) {
          reject("ERR_MAX_RETRIES");
        } else {
          const exponentialDelayMs = calculateExponentialDelay(numRetries);

          console.log("Waiting", exponentialDelayMs, "ms");
          setTimeout(async () => {
            // Here, we resume the sequence with a new connection
            const result = await initiateConnection(
              initialNumber,
              n - numReceived,
              m + numReceived,
              numRetries
            );
            resolve(result);
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

  const sum = await initiateConnection();

  console.log("Sum found", sum);
}
