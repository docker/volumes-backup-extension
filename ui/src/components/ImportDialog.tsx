import React, { useContext } from "react";
import {
  Button,
  Typography,
  Grid,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function ImportDialog({ ...props }) {
  console.log("ImportDialog component rendered.");
  const ddClient = useDockerDesktopClient();

  const context = useContext(MyContext);
  const fileName = `${context.store.volumeName}.tar.gz`;

  const [path, setPath] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

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

  const importVolume = async () => {
    setActionInProgress(true);

    try {
      let hostPath = path;

      if (props.dockerContextName !== "default") {
        // create container in remote where to copy the file
        console.log(
          "Creating container in remote host to copy the file from local filesystem..."
        );
        const runOutput = await ddClient.docker.cli.exec("--context", [
          `${props.dockerContextName}`,
          "run",
          `--name=copy-to-remote-ctr`,
          "-d",
          "busybox",
          "sleep",
          "120",
        ]);
        if (runOutput.stderr !== "") {
          ddClient.desktopUI.toast.error(runOutput.stderr);
          return;
        }

        console.log(
          `Copying file ${path} to copy-to-remote-ctr:/tmp using docker context ${props.dockerContextName}`
        );

        const cpOutput = await ddClient.docker.cli.exec("--context", [
          `${props.dockerContextName}`,
          "cp",
          `${path}`,
          `copy-to-remote-ctr:/tmp`,
        ]);
        if (cpOutput.stderr !== "") {
          ddClient.desktopUI.toast.error(cpOutput.stderr);
          return;
        }

        const filename = path.split("/").pop();
        hostPath = `/tmp/${filename}`;
      }

      console.log("Importing file into volume...");
      const args = [
        `${props.dockerContextName}`,
        "run",
        "--rm",
        `-v=${context.store.volumeName}:/vackup-volume `,
        `-v=${hostPath}:/vackup`, // path: e.g. "$HOME/Downloads/my-vol.tar.gz"
        "busybox",
        "tar",
        "-xvzf",
        `/vackup`,
      ];
      console.log(args.join(" "));
      const output = await ddClient.docker.cli.exec("--context", args);
      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
        return;
      }
      ddClient.desktopUI.toast.success(
        `File ${fileName} imported into volume ${context.store.volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to import file ${fileName} into volume ${context.store.volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      if (props.dockerContextName !== "default") {
        console.log("Removing temp remote container...");
        const args = [
          props.dockerContextName,
          "rm",
          "-f",
          "copy-to-remote-ctr",
        ];
        console.log(args.join(" "));
        const output = await ddClient.docker.cli.exec("--context", args);
        console.log(output);
      }

      setActionInProgress(false);
      setPath("");
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Import gzip'ed tarball into a volume</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>
          Extracts a gzip'ed tarball into a volume.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
              Choose a .tar.gz to import, e.g. file.tar.gz
            </Typography>
          </Grid>
          <Grid item>
            <Button variant="contained" onClick={selectImportTarGzFile}>
              Select file
            </Button>
          </Grid>

          {path !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                The file {path} will be imported into volume{" "}
                {context.store.volumeName}.
              </Typography>
              <Typography variant="body1" color="text.secondary">
                ⚠️ This will replace all the existing data inside the volume.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => {
            setPath("");
            props.onClose();
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={importVolume}
          disabled={path === "" || fileName === ""}
        >
          Import
        </Button>
      </DialogActions>
    </Dialog>
  );
}
