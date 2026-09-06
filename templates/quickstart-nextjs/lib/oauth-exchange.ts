import { exchangeToken } from "./api";
import {
  clearReferralCode,
  readReferralCode,
  readSession,
  writeSession,
  type AuthSession
} from "./auth";
import { clearOAuthVerifier, readOAuthReturnTo, readOAuthVerifier, type OAuthReturnTo } from "./oauth-flow";

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

type OAuthSessionResult = { session: AuthSession; returnTo: OAuthReturnTo };

const oauthExchangeCoordinator = createOAuthExchangeCoordinator<OAuthSessionResult>({
  exchange: async (code, flowID) => {
    const returnTo = readOAuthReturnTo(flowID);
    const session = await exchangeToken(code, readReferralCode(), readOAuthVerifier(flowID));
    clearOAuthVerifier(flowID);
    // Keep the destination with the completed single-flight result before its
    // one-time browser record is cleared. React effect replays use this snapshot.
    return { session, returnTo };
  },
  readSessionToken: () => readSession()?.token || "",
  commit: ({ session }) => {
    clearReferralCode();
    writeSession(session);
  }
});

export function exchangeOAuthCodeOnce(
  code: string,
  flowID = ""
): Promise<OAuthExchangeOutcome<AuthSession> & { returnTo: OAuthReturnTo }> {
  return oauthExchangeCoordinator.exchange(code, flowID).then(({ session: result, committed }) => ({
    session: result.session,
    returnTo: result.returnTo,
    committed
  }));
}
