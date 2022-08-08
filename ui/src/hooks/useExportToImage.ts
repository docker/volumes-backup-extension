import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useContext, useState } from "react";
import { MyContext } from "..";

const ddClient = createDockerDesktopClient();

export const useExportToImage = () => {
  const [isLoading, setIsLoading] = useState(false);
  const context = useContext(MyContext);
  const selectedVolumeName = context.store.volume?.volumeName;

  const exportToImage = ({
    imageName,
  }: {
    imageName: string;
  }) => {
    setIsLoading(true);

    return ddClient.extension.vm.service
    .get(`/volumes/${context.store.volume.volumeName}/save?image=${imageName}`)
      .then((_: any) => {
        ddClient.desktopUI.toast.success(
          `Volume ${selectedVolumeName} exported to ${imageName}`
        );
      })
      .catch((error) => {
        ddClient.desktopUI.toast.error(
          `Failed to backup volume ${selectedVolumeName} to ${imageName}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    exportToImage,
    isLoading,
  };
};
