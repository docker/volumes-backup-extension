import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useState } from "react";
const ddClient = createDockerDesktopClient();

interface ILoadImage {
  volumeName: string;
  imageName: string;
}
export const useImportFromImage = () => {
  const [isInProgress, setIsInProgress] = useState(false);

  const loadImage = async ({ volumeName, imageName }: ILoadImage) => {
    setIsInProgress(true);

    return ddClient.extension.vm.service
      .get(`/volumes/${volumeName}/load?image=${imageName}`)
      .then((_: any) => {
        setIsInProgress(false);
        ddClient.desktopUI.toast.success(
          `Copied /volume-data from image ${imageName} into volume ${volumeName}`
        );
      })
      .catch((error) => {
        setIsInProgress(false);
        ddClient.desktopUI.toast.error(
          `Failed to copy /volume-data from image ${imageName} to into volume ${volumeName}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      });
  };

  return {
    loadImage,
    isInProgress,
  };
};
