import React, { useEffect } from "react";
import {
  DataGrid,
  GridCellParams,
  GridActionsCellItem,
} from "@mui/x-data-grid";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import {
  Stack,
  Button,
  Typography,
  Box,
  LinearProgress,
  Grid,
} from "@mui/material";
import {
  Download as DownloadIcon,
  Upload as UploadIcon,
} from "@mui/icons-material";

const sleep = (milliseconds) => {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
};

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export function App() {
  const [rows, setRows] = React.useState([]);
  const [volumeContainersMap, setVolumeContainersMap] = React.useState<
    Record<string, string[]>
  >({});
  const [volumes, setVolumes] = React.useState([]);
  const [path, setPath] = React.useState<string>("");
  const [reloadTable, setReloadTable] = React.useState<boolean>(false);
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);
  const ddClient = useDockerDesktopClient();

  const columns = [
    { field: "id", headerName: "ID", width: 70, hide: true },
    { field: "volumeDriver", headerName: "Driver", width: 70 },
    {
      field: "volumeName",
      headerName: "Volume name",
      width: 320,
    },
    { field: "volumeLinks", hide: true },
    {
      field: "volumeContainers",
      headerName: "Containers",
      width: 260,
      renderCell: (params) => {
        if (params.row.volumeContainers) {
          return (
            <Box display="flex" flexDirection="column">
              {params.row.volumeContainers.map((container) => (
                <Typography key={container}>{container}</Typography>
              ))}
            </Box>
          );
        }
        return <></>;
      },
    },
    { field: "volumeSize", headerName: "Size", width: 130 },
    {
      field: "actions",
      type: "actions",
      width: 130,
      sortable: false,
      getActions: (params) => [
        <GridActionsCellItem
          key={"action_export_" + params.row.id}
          icon={<DownloadIcon>Export</DownloadIcon>}
          label="Export"
          onClick={handleExport(params.row)}
          disabled={path === "" || actionInProgress}
        />,
        <GridActionsCellItem
          key={"action_import_" + params.row.id}
          icon={<UploadIcon>Import</UploadIcon>}
          label="Import"
          onClick={handleImport(params.row)}
          disabled={path === "" || actionInProgress}
        />,
      ],
    },
  ];

  const handleExport = (row) => () => {
    exportVolume(row.volumeName);
  };

  const handleImport = (row) => () => {
    importVolume(row.volumeName);

    // hack to reduce the likelihood of having "another disk operation is already running"
    sleep(1000).then(() => {
      setReloadTable(!reloadTable);
    });
  };

  const handleCellClick = (params: GridCellParams) => {
    if (params.colDef.field === "volumeName") {
      ddClient.desktopUI.navigate.viewVolume(params.row.volumeName);
    }
  };

  useEffect(() => {
    const listVolumes = async () => {
      try {
        const result = await ddClient.docker.cli.exec("system", [
          "df",
          "-v",
          "--format",
          "'{{ json .Volumes }}'",
        ]);

        if (result.stderr !== "") {
          ddClient.desktopUI.toast.error(result.stderr);
        } else {
          const volumes = result.parseJsonObject();

          volumes.forEach((volume) => {
            getContainersForVolume(volume.Name).then((containers) => {
              setVolumeContainersMap((current) => {
                const next = { ...current };
                next[volume.Name] = containers;
                return next;
              });
            });
          });

          setVolumes(volumes);
        }
      } catch (error) {
        ddClient.desktopUI.toast.error(
          `Failed to list volumes: ${error.stderr}`
        );
      }
    };

    listVolumes();

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reloadTable]);

  useEffect(() => {
    const rows = volumes
      .sort((a, b) => a.Name.localeCompare(b.Name))
      .map((volume, index) => {
        return {
          id: index,
          volumeDriver: volume.Driver,
          volumeName: volume.Name,
          volumeLinks: volume.Links,
          volumeContainers: volumeContainersMap[volume.Name],
          volumeSize: volume.Size,
        };
      });

    setRows(rows);
  }, [volumeContainersMap]);

  const selectExportDirectory = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openDirectory"],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const selectImportTarGzFile = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openFile"],
        filters: [{ name: ".tar.gz", extensions: [".tar.gz"] }],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const exportVolume = async (volumeName: string) => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${volumeName}:/vackup-volume `,
        `-v=${path}:/vackup `,
        "busybox",
        "tar",
        "-zcvf",
        `/vackup/${volumeName}.tar.gz`,
        "/vackup-volume",
      ]);
      if (output.stderr !== "") {
        //"tar: removing leading '/' from member names\n"
        if (!output.stderr.includes("tar: removing leading")) {
          // this is an error we may want to display
          ddClient.desktopUI.toast.error(output.stderr);
          return;
        }
      }
      ddClient.desktopUI.toast.success(
        `Volume ${volumeName} exported to ${path}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to backup volume ${volumeName} to ${path}: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
    }
  };

  const importVolume = async (volumeName: string) => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${volumeName}:/vackup-volume `,
        `-v=${path}:/vackup `, // e.g. "$HOME/Downloads/my-vol.tar.gz"
        "busybox",
        "tar",
        "-xvzf",
        `/vackup`,
      ]);
      console.log(output);
      if (output.stderr !== "") {
        //"tar: removing leading '/' from member names\n"
        if (!output.stderr.includes("tar: removing leading")) {
          // this is an error we may want to display
          ddClient.desktopUI.toast.error(output.stderr);
          return;
        }
      }
      ddClient.desktopUI.toast.success(
        `File ${volumeName}.tar.gz imported into volume ${volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to import file ${volumeName}.tar.gz into volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
    }
  };

  const getContainersForVolume = async (
    volumeName: string
  ): Promise<string[]> => {
    try {
      const output = await ddClient.docker.cli.exec("ps", [
        "-a",
        `--filter="volume=${volumeName}"`,
        `--format='{{ .Names}}'`,
      ]);

      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
      }

      return output.stdout.trim().split(" ");
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to get containers for volume ${volumeName}: ${error.stderr} Error code: ${error.code}`
      );
    }
  };

  return (
    <>
      <Typography variant="h3">Vackup Extension</Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
        Easily backup and restore docker volumes.
      </Typography>
      <Stack direction="column" alignItems="start" spacing={2} sx={{ mt: 4 }}>
        <Grid
          container
          spacing={2}
          justifyContent="center"
          textAlign="center"
          alignItems="center"
        >
          <Grid item>
            <Button
              variant="contained"
              onClick={selectExportDirectory}
              disabled={actionInProgress}
            >
              Choose a path to export
            </Button>
          </Grid>
          <Grid item>or</Grid>
          <Grid item>
            <Button
              variant="contained"
              onClick={selectImportTarGzFile}
              disabled={actionInProgress}
            >
              Choose a .tar.gz to import
            </Button>
          </Grid>
        </Grid>
        <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
          {path}
        </Typography>
        {actionInProgress && (
          <Box sx={{ width: "100%" }}>
            <LinearProgress />
          </Box>
        )}

        <Box width="100%">
          <DataGrid
            rows={rows}
            columns={columns}
            pageSize={5}
            rowsPerPageOptions={[5]}
            checkboxSelection={false}
            disableSelectionOnClick={true}
            autoHeight
            getRowHeight={() => "auto"}
            onCellClick={handleCellClick}
            sx={{
              "&.MuiDataGrid-root--densityCompact .MuiDataGrid-cell": { py: 1 },
              "&.MuiDataGrid-root--densityStandard .MuiDataGrid-cell": {
                py: 1,
              },
              "&.MuiDataGrid-root--densityComfortable .MuiDataGrid-cell": {
                py: 2,
              },
            }}
          />
        </Box>
      </Stack>
    </>
  );
}
