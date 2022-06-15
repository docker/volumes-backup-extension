import React, { useEffect } from "react";
import { DataGrid } from "@mui/x-data-grid";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { Stack, Button, Typography } from "@mui/material";

// Note: This line relies on Docker Desktop's presence as a host application.
// If you're running this React app in a browser, it won't work properly.
const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export function App() {
  const [rows, setRows] = React.useState([]);
  const [exportPath, setExportPath] = React.useState<string>("");
  const ddClient = useDockerDesktopClient();

  const columns = [
    { field: "id", headerName: "ID", width: 70, hide: true },
    { field: "volumeDriver", headerName: "Driver", width: 70 },
    { field: "volumeName", headerName: "Volume name", width: 260 },
    { field: "volumeMountPoint", headerName: "Mount point", width: 260 },
    { field: "volumeSize", headerName: "Size", width: 130 },
    {
      field: "exportPath",
      headerName: "Export path",
      width: 130,
      renderCell: (params) => {
        const onClick = (e) => {
          e.stopPropagation(); // don't select this row after clicking
          selectExportDirectory();
        };

        return (
          <Button variant="contained" onClick={onClick}>
            Choose path
          </Button>
        );
      },
    },
    {
      field: "export",
      headerName: "Action",
      width: 130,
      renderCell: (params) => {
        const onClick = (e) => {
          e.stopPropagation(); // don't select this row after clicking

          console.log("exporting volume", params.row.volumeName);
          console.log(params);
          exportVolume(params.row.volumeName);
        };

        return (
          <Button
            variant="contained"
            onClick={onClick}
            disabled={exportPath === ""}
          >
            Export
          </Button>
        );
      },
    },
  ];

  useEffect(() => {
    const listVolumes = async () => {
      const result = await ddClient.docker.cli.exec("volume", [
        "ls",
        "--format",
        "'{{ json . }}'",
      ]);

      if (result.stderr !== "") {
        ddClient.desktopUI.toast.error(result.stderr);
      } else {
        const volumes = result.parseJsonLines();
        const rows = volumes.map((volume, index) => {
          return {
            id: index,
            volumeDriver: volume.Driver,
            volumeName: volume.Name,
            volumeMountPoint: volume.Mountpoint,
            volumeSize: volume.Size,
          };
        });

        setRows(rows);
      }
    };

    listVolumes();
  }, []); // run it once, only when component is mounted

  const selectExportDirectory = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({ properties: ["openDirectory"] })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        console.log("export path", result.filePaths[0]);
        setExportPath(result.filePaths[0]);
      });
  };

  const exportVolume = async (volumeName: string) => {
    console.log("export");

    const filename = "backup.tar.gz";

    const output = await ddClient.docker.cli.exec("run", [
      "--rm",
      `-v=${volumeName}:/vackup-volume `,
      `-v=${exportPath}:/vackup `,
      "busybox",
      "tar",
      "-zcvf",
      `/vackup/${filename}`,
      "/vackup-volume",
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
      `Volume ${volumeName} exported to ${exportPath}`
    );
  };

  return (
    <>
      <Typography variant="h3">Vackup Extension</Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
        Easily backup and restore docker volumes.
      </Typography>
      <Stack direction="row" alignItems="start" spacing={2} sx={{ mt: 4 }}>
        <div style={{ height: 400, width: "100%" }}>
          <DataGrid
            rows={rows}
            columns={columns}
            pageSize={5}
            rowsPerPageOptions={[5]}
            checkboxSelection={false}
          />
        </div>
      </Stack>
    </>
  );
}
