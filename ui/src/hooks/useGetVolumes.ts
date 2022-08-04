import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useEffect, useState } from "react";

const ddClient = createDockerDesktopClient();

type VolumeData = {
  Driver: string;
  Size: string;
  Containers: string[];
};

interface IRow {
  id: number;
  volumeDriver: string;
  volumeName: string;
  volumeContainers: unknown[] | null;
  volumeSize: string;
}

export const useGetVolumes = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [data, setData] = useState<IRow[]>();

  useEffect(() => {
    listVolumes();
  }, []);

  const listVolumes = async () => {
    const startTime = performance.now();
    setIsLoading(true);

    try {
      ddClient.extension.vm.service
        .get("/volumes")
        .then((results: Record<string, VolumeData>) => {
          let rows: IRow[] = [];
          let index = 0;
          for (const key in results) {
            const value = results[key];
            rows.push({
              id: index,
              volumeDriver: value.Driver,
              volumeName: key,
              volumeContainers: value.Containers.length
                ? value.Containers
                : null,
              volumeSize: value.Size,
            });
            index++;
          }

          setIsLoading(false);
          const endTime = performance.now();
          console.log(`[listVolumes] took ${endTime - startTime} ms.`);
          setData(rows);
        });
    } catch (error) {
      setIsLoading(false);
      ddClient.desktopUI.toast.error(`Failed to list volumes: ${error.stderr}`);
    }
  };

  return {
    listVolumes,
    isLoading,
    data,
    setData,
  };
};
