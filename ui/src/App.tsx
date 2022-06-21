import React, { useEffect, useContext } from "react";
import {
  DataGrid,
  GridCellParams,
  GridActionsCellItem,
} from "@mui/x-data-grid";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import {
  Backdrop,
  Box,
  CircularProgress,
  LinearProgress,
  Stack,
  Tooltip,
  Typography,
  Grid,
} from "@mui/material";
import {
  Download as DownloadIcon,
  Upload as UploadIcon,
  Delete as DeleteIcon,
  Layers as LayersIcon,
  ArrowCircleDown as ArrowCircleDownIcon,
  ExitToApp as ExitToAppIcon,
} from "@mui/icons-material";
import ExportDialog from "./components/ExportDialog";
import ImportDialog from "./components/ImportDialog";
import SaveDialog from "./components/SaveDialog";
import LoadDialog from "./components/LoadDialog";
import { MyContext } from ".";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export function App() {
  const context = useContext(MyContext);
  const [rows, setRows] = React.useState([]);
  const [volumeContainersMap, setVolumeContainersMap] = React.useState<
    Record<string, string[]>
  >({});
  const [volumes, setVolumes] = React.useState([]);
  const [reloadTable, setReloadTable] = React.useState<boolean>(false);
  const [loadingVolumes, setLoadingVolumes] = React.useState<boolean>(true);

  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const [openExportDialog, setOpenExportDialog] =
    React.useState<boolean>(false);
  const [openImportDialog, setOpenImportDialog] =
    React.useState<boolean>(false);
  const [openSaveDialog, setOpenSaveDialog] = React.useState<boolean>(false);
  const [openLoadDialog, setOpenLoadDialog] = React.useState<boolean>(false);

  const ddClient = useDockerDesktopClient();

  const columns = [
    { field: "id", headerName: "ID", width: 70, hide: true },
    { field: "volumeDriver", headerName: "Driver" },
    {
      field: "volumeName",
      headerName: "Volume name",
      flex: 1,
    },
    { field: "volumeLinks", hide: true },
    {
      field: "volumeContainers",
      headerName: "Containers",
      flex: 1,
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
    { field: "volumeSize", headerName: "Size" },
    {
      field: "actions",
      type: "actions",
      headerName: "Actions",
      minWidth: 200,
      sortable: false,
      getActions: (params) => [
        <GridActionsCellItem
          key={"action_view_volume_" + params.row.id}
          icon={
            <Tooltip title="View volume">
              <ExitToAppIcon>View volume</ExitToAppIcon>
            </Tooltip>
          }
          label="View volume"
          onClick={handleNavigate(params.row)}
          disabled={actionInProgress}
        />,
        <GridActionsCellItem
          key={"action_export_" + params.row.id}
          icon={
            <Tooltip title="Export volume">
              <DownloadIcon>Export volume</DownloadIcon>
            </Tooltip>
          }
          label="Export volume"
          onClick={handleExport(params.row)}
        />,
        <GridActionsCellItem
          key={"action_import_" + params.row.id}
          icon={
            <Tooltip title="Import gzip'ed tarball">
              <UploadIcon>Import gzip'ed tarball</UploadIcon>
            </Tooltip>
          }
          label="Import gzip'ed tarball"
          onClick={handleImport(params.row)}
        />,
        <GridActionsCellItem
          key={"action_save_" + params.row.id}
          icon={
            <Tooltip title="Save to image">
              <LayersIcon>Save to image</LayersIcon>
            </Tooltip>
          }
          label="Save to image"
          onClick={handleSave(params.row)}
        />,
        <GridActionsCellItem
          key={"action_load_" + params.row.id}
          icon={
            <Tooltip title="Load from image">
              <ArrowCircleDownIcon>Load from image</ArrowCircleDownIcon>
            </Tooltip>
          }
          label="Load from image"
          onClick={handleLoad(params.row)}
        />,
        <GridActionsCellItem
          key={"action_empty_" + params.row.id}
          icon={
            <Tooltip title="Empty volume">
              <DeleteIcon>Empty volume</DeleteIcon>
            </Tooltip>
          }
          label="Empty volume"
          onClick={handleEmpty(params.row)}
        />,
      ],
    },
  ];

  const handleNavigate = (row) => async () => {
    ddClient.desktopUI.navigate.viewVolume(row.volumeName);
  };

  const handleExport = (row) => () => {
    setOpenExportDialog(true);
    context.actions.setVolumeName(row.volumeName);
  };

  const handleImport = (row) => () => {
    setOpenImportDialog(true);
    context.actions.setVolumeName(row.volumeName);
  };

  const handleSave = (row) => () => {
    setOpenSaveDialog(true);
    context.actions.setVolumeName(row.volumeName);
  };

  const handleLoad = (row) => async () => {
    setOpenLoadDialog(true);
    context.actions.setVolumeName(row.volumeName);
  };

  const handleEmpty = (row) => async () => {
    await emptyVolume(row.volumeName);
    setReloadTable(!reloadTable);
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
          '"{{ json .Volumes }}"',
        ]);

        if (result.stderr !== "") {
          ddClient.desktopUI.toast.error(result.stderr);
        } else {
          const volumes = result.parseJsonObject();

          const promises = volumes.map((volume) =>
            getContainersForVolume(volume.Name)
          );

          Promise.all(promises)
            .then((values) => {
              const map = {};
              values.map(({ volumeName, containers }) => {
                return (map[volumeName] = containers);
              });

              setVolumeContainersMap(map);
            })
            .finally(() => {
              setLoadingVolumes(false);
            });

          setVolumes(volumes);
        }
      } catch (error) {
        setLoadingVolumes(false);
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

  const emptyVolume = async (volumeName: string) => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${volumeName}:/vackup-volume `,
        "busybox",
        "/bin/sh",
        "-c",
        '"rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/*"', // hidden and not-hidden files and folders: .[!.]* matches all dot files except . and files whose name begins with .., and ..?* matches all dot-dot files except ..
      ]);
      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
        return;
      }
      ddClient.desktopUI.toast.success(
        `The content of volume ${volumeName} has been removed`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to empty volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
    }
  };

  const getContainersForVolume = async (
    volumeName: string
  ): Promise<{ volumeName: string; containers: string[] }> => {
    try {
      const output = await ddClient.docker.cli.exec("ps", [
        "-a",
        `--filter="volume=${volumeName}"`,
        `--format="{{ .Names}}"`,
      ]);

      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
      }

      return { volumeName, containers: output.stdout.trim().split(" ") };
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to get containers for volume ${volumeName}: ${error.stderr} Error code: ${error.code}`
      );
    }
  };

  const handleExportDialogClose = () => {
    setOpenExportDialog(false);
  };

  const handleImportDialogClose = () => {
    setOpenImportDialog(false);
    setReloadTable(!reloadTable);
  };

  const handleSaveDialogClose = () => {
    setOpenSaveDialog(false);
  };

  const handleLoadDialogClose = () => {
    setOpenLoadDialog(false);
    setReloadTable(!reloadTable);
  };

  return (
    <>
      <Typography variant="h3">Vackup Extension</Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
        Easily backup and restore docker volumes.
      </Typography>
      <Stack direction="column" alignItems="start" spacing={2} sx={{ mt: 4 }}>
        <Grid container>
          <Grid item flex={1}>
            <Backdrop
              sx={{
                backgroundColor: "rgba(245,244,244,0.4)",
                zIndex: (theme) => theme.zIndex.drawer + 1,
              }}
              open={actionInProgress}
            >
              <CircularProgress color="info" />
            </Backdrop>
            <DataGrid
              loading={loadingVolumes}
              components={{
                LoadingOverlay: LinearProgress,
              }}
              rows={rows}
              columns={columns}
              pageSize={10}
              rowsPerPageOptions={[10]}
              checkboxSelection={false}
              disableSelectionOnClick={true}
              autoHeight
              getRowHeight={() => "auto"}
              onCellClick={handleCellClick}
              sx={{
                "&.MuiDataGrid-root--densityCompact .MuiDataGrid-cell": {
                  py: 1,
                },
                "&.MuiDataGrid-root--densityStandard .MuiDataGrid-cell": {
                  py: 1,
                },
                "&.MuiDataGrid-root--densityComfortable .MuiDataGrid-cell": {
                  py: 2,
                },
              }}
            />
          </Grid>

          {openExportDialog && (
            <ExportDialog
              open={openExportDialog}
              onClose={handleExportDialogClose}
            />
          )}

          {openImportDialog && (
            <ImportDialog
              open={openImportDialog}
              onClose={handleImportDialogClose}
            />
          )}

          {openSaveDialog && (
            <SaveDialog open={openSaveDialog} onClose={handleSaveDialogClose} />
          )}

          {openLoadDialog && (
            <LoadDialog open={openLoadDialog} onClose={handleLoadDialogClose} />
          )}
        </Grid>
      </Stack>
    </>
  );
}
