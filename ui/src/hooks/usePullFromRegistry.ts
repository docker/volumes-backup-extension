import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useContext, useState } from "react";
import { MyContext } from "..";

const ddClient = createDockerDesktopClient();

export const usePullFromRegistry = () => {
  const [isLoading, setIsLoading] = useState(false);
  const context = useContext(MyContext);

  const pullFromRegistry = ({ imageName }: { imageName: string }) => {
    setIsLoading(true);

    return ddClient.extension.host.cli
      .exec("volumes-share-client", [
        "--extension-dir",
        process.env["REACT_APP_EXTENSION_INSTALLATION_DIR_NAME"],
        "pull",
        imageName,
        context.store.volume.volumeName,
      ])
      .then((result) => {
        ddClient.desktopUI.toast.success(
          `Volume ${context.store.volume.volumeName} pulled as ${imageName} from registry`
        );
      })
      .catch((error) => {
        console.error(error);
        ddClient.desktopUI.toast.error(
          `Failed to pull volume ${context.store.volume.volumeName} as ${imageName} from registry: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    pullFromRegistry,
    isLoading,
  };
};
