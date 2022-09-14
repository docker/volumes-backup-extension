import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useEffect, useState } from "react";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

type VolumeData = {
  Driver: string;
  Size: number;
  SizeHuman: string;
  Containers: string[];
};

export interface IVolumeRow {
  id: number;
  volumeDriver: string;
  volumeName: string;
  volumeContainers?: unknown[] | null;
  volumeSize?: string;
  volumeBytes?: number;
}

export const useGetVolumes = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [isVolumesSizeLoading, setIsVolumesSizeLoading] = useState(false);
  const [data, setData] = useState<IVolumeRow[]>();
  const { sendNotification } = useNotificationContext();

  useEffect(() => {
    listVolumes();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const listVolumes = async () => {
    const startTime = performance.now();
    setIsLoading(true);

    try {
      ddClient.extension.vm.service
        .get("/volumes")
        .then((results: Record<string, VolumeData>) => {
          const rows: IVolumeRow[] = [];
          let index = 0;
          for (const key in results) {
            const value = results[key];
            rows.push({
              id: index,
              volumeName: key,
              volumeDriver: value.Driver,
            });
            index++;
          }

          setIsLoading(false);
          const endTime = performance.now();
          console.log(`[listVolumes] took ${endTime - startTime} ms.`);
          setData(rows);

          setIsVolumesSizeLoading(true);
          const fetchVolumesSize = new Promise<any>((resolve) => {
            ddClient.extension.vm.service
              .get("/volumes/size")
              .then((results: Record<string, string>) => {
                resolve(results);
              });
          });

          const fetchVolumesContainer = new Promise<any>((resolve) => {
            ddClient.extension.vm.service
              .get("/volumes/container")
              .then((results: Record<string, string>) => {
                resolve(results);
              });
          });

          // Fetch volumes size and containers attached
          Promise.all([fetchVolumesSize, fetchVolumesContainer]).then(
            (values) => {
              const sizesMap = values[0];
              const containersMap = values[1];

              const updatedRows: IVolumeRow[] = [];
              for (const key in rows) {
                const row = rows[key];

                if (containersMap[row.volumeName] !== undefined) {
                  if (containersMap[row.volumeName].Containers?.length) {
                    row.volumeContainers =
                      containersMap[row.volumeName].Containers;
                  }
                }

                if (sizesMap[row.volumeName] !== undefined) {
                  row.volumeSize = sizesMap[row.volumeName].Human;
                  row.volumeBytes = sizesMap[row.volumeName].Bytes;
                }

                updatedRows.push(row);
              }

              setData(updatedRows);
              setIsVolumesSizeLoading(false);
            }
          );
        });
    } catch (error) {
      setIsLoading(false);
      setIsVolumesSizeLoading(false);
      sendNotification.error(`Failed to list volumes: ${error.stderr}`);
    }
  };

  return {
    listVolumes,
    isLoading,
    isVolumesSizeLoading,
    data,
    setData,
  };
};
