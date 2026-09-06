import { exchangeToken } from "./api";
import {
  clearReferralCode,
  readReferralCode,
  readSession,
  writeSession,
  type AuthSession
} from "./auth";
import { clearOAuthVerifier, readOAuthVerifier } from "./oauth-flow";

export type OAuthExchangeOutcome<TSession> = {
  session: TSession;
  committed: boolean;
};

type OAuthExchangeFlight<TSession> = {
  code: string;
  flowID: string;
  initialSessionToken: string;
  request: Promise<OAuthExchangeOutcome<TSession>>;
};

type OAuthExchangeCoordinatorDeps<TSession> = {
  exchange: (code: string, flowID: string) => Promise<TSession>;
  readSessionToken: () => string;
  commit: (session: TSession) => void;
};

export function createOAuthExchangeCoordinator<TSession>(
  deps: OAuthExchangeCoordinatorDeps<TSession>
) {
  let currentFlight: OAuthExchangeFlight<TSession> | null = null;

  return {
    exchange(code: string, flowID = ""): Promise<OAuthExchangeOutcome<TSession>> {
      if (currentFlight?.code === code && currentFlight.flowID === flowID) {
        return currentFlight.request;
      }

      const initialSessionToken = deps.readSessionToken();
      let flight: OAuthExchangeFlight<TSession>;
      const request = deps.exchange(code, flowID)
        .then((session) => {
          const committed = currentFlight === flight
            && deps.readSessionToken() === initialSessionToken;
          if (committed) {
            deps.commit(session);
          }
          return { session, committed };
        })
        .catch((error) => {
          if (currentFlight === flight) {
            currentFlight = null;
          }
          throw error;
        });
      flight = { code, flowID, initialSessionToken, request };
      currentFlight = flight;
      return request;
    }
  };
}

const oauthExchangeCoordinator = createOAuthExchangeCoordinator<AuthSession>({
  exchange: async (code, flowID) => {
    const session = await exchangeToken(code, readReferralCode(), readOAuthVerifier(flowID));
    clearOAuthVerifier(flowID);
    return session;
  },
  readSessionToken: () => readSession()?.token || "",
  commit: (session) => {
    clearReferralCode();
    writeSession(session);
  }
});

export function exchangeOAuthCodeOnce(
  code: string,
  flowID = ""
): Promise<OAuthExchangeOutcome<AuthSession>> {
  return oauthExchangeCoordinator.exchange(code, flowID);
}
